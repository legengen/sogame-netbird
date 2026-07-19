package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	clientnetbird "github.com/legengen/sogame-netbird/client/internal/netbird"
	"github.com/legengen/sogame-netbird/client/internal/roomapi"
	"github.com/legengen/sogame-netbird/client/internal/securestore"
)

const cleanupTimeout = 5 * time.Second

var (
	ErrRoomAlreadySaved    = errors.New("a room is already saved")
	ErrStoredStateConflict = errors.New("saved room state is incomplete or inconsistent")
	ErrCommandInProgress   = errors.New("a room command is already in progress")
)

type RoomAPI interface {
	Create(context.Context, *roomapi.CreateIntent) (roomapi.Enrollment, error)
	Join(context.Context, string) (roomapi.Enrollment, error)
}

type MetadataStorage interface {
	Load() (securestore.RoomMetadata, error)
	Save(securestore.RoomMetadata) error
	Clear() error
}

type RoomCodeStorage interface {
	Load() ([]byte, error)
	Save([]byte) error
	Clear() error
}

type TransactionError struct {
	Cause           error
	CleanupFailures int
}

func (e *TransactionError) Error() string {
	if e.CleanupFailures == 0 {
		return "room enrollment transaction failed"
	}
	return fmt.Sprintf("room enrollment transaction failed; %d cleanup operations failed", e.CleanupFailures)
}

func (e *TransactionError) Unwrap() error { return e.Cause }

type Service struct {
	rooms    RoomAPI
	netbird  clientnetbird.Adapter
	metadata MetadataStorage
	codes    RoomCodeStorage
	machine  *Machine
	now      func() time.Time
	mu       sync.Mutex
	busy     bool
}

func NewService(rooms RoomAPI, netbird clientnetbird.Adapter, metadata MetadataStorage, codes RoomCodeStorage) *Service {
	return &Service{
		rooms:    rooms,
		netbird:  netbird,
		metadata: metadata,
		codes:    codes,
		machine:  NewMachine(),
		now:      time.Now,
	}
}

func (s *Service) State() Snapshot { return s.machine.Snapshot() }

func (s *Service) Create(ctx context.Context, hostname string) (Snapshot, error) {
	intent, err := roomapi.NewCreateIntent()
	if err != nil {
		return s.fail(err)
	}
	return s.enroll(ctx, hostname, func(ctx context.Context) (roomapi.Enrollment, error) {
		return s.rooms.Create(ctx, intent)
	})
}

func (s *Service) Join(ctx context.Context, roomCode, hostname string) (Snapshot, error) {
	return s.enroll(ctx, hostname, func(ctx context.Context) (roomapi.Enrollment, error) {
		return s.rooms.Join(ctx, roomCode)
	})
}

func (s *Service) enroll(ctx context.Context, hostname string, obtain func(context.Context) (roomapi.Enrollment, error)) (Snapshot, error) {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" || len(hostname) > 63 {
		return s.fail(errors.New("device name must contain 1 to 63 characters"))
	}
	if err := s.beginCommand(); err != nil {
		return s.State(), err
	}
	defer s.endCommand()
	if err := s.requireEmptyStorage(); err != nil {
		return s.fail(err)
	}
	s.machine.Apply(Facts{EnrollmentInProgress: true})

	enrollment, err := obtain(ctx)
	if err != nil {
		return s.fail(err)
	}
	defer enrollment.DiscardSetupKey()
	profile, err := s.netbird.CreateProfile(ctx, clientnetbird.ManagedProfileName)
	if err != nil {
		return s.fail(&TransactionError{Cause: err})
	}
	transaction := enrollmentTransaction{
		service:   s,
		profileID: profile.ID,
	}
	if profile.ID == "" {
		return s.fail(&TransactionError{Cause: clientnetbird.ErrManagedProfileInconsistent})
	}
	if profile.Name != clientnetbird.ManagedProfileName {
		return s.fail(transaction.wrap(clientnetbird.ErrManagedProfileInconsistent, ctx))
	}
	committed := false
	defer func() {
		if !committed {
			transaction.compensate(ctx)
		}
	}()

	transaction.enrollmentAttempted = true
	err = enrollment.ConsumeSetupKey(func(key *clientnetbird.SetupKey) error {
		return s.netbird.Enroll(ctx, clientnetbird.EnrollmentRequest{
			ManagementURL: enrollment.ManagementURL,
			ProfileID:     profile.ID,
			Hostname:      hostname,
			SetupKey:      key,
		})
	})
	if err != nil {
		return s.fail(transaction.wrap(err, ctx))
	}
	if err := s.netbird.Connect(ctx, profile.ID); err != nil {
		return s.fail(transaction.wrap(err, ctx))
	}

	roomCode := []byte(enrollment.RoomCode)
	defer clearBytes(roomCode)
	transaction.codeWriteAttempted = true
	if err := s.codes.Save(roomCode); err != nil {
		return s.fail(transaction.wrap(err, ctx))
	}
	transaction.metadataWriteAttempted = true
	if err := s.metadata.Save(securestore.RoomMetadata{
		Version:       securestore.CurrentMetadataVersion,
		RoomID:        enrollment.RoomID,
		ManagementURL: enrollment.ManagementURL,
		ProfileID:     profile.ID,
		CreatedAt:     s.now().UTC(),
	}); err != nil {
		return s.fail(transaction.wrap(err, ctx))
	}
	committed = true

	facts := Facts{RoomSaved: true}
	if status, statusErr := s.netbird.Status(ctx); statusErr == nil {
		facts.ControlPlaneReady = status.ManagementConnected && status.SignalConnected
		facts.DaemonPeers = status.Peers
	}
	return s.machine.Apply(facts), nil
}

func (s *Service) beginCommand() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return ErrCommandInProgress
	}
	s.busy = true
	return nil
}

func (s *Service) endCommand() {
	s.mu.Lock()
	s.busy = false
	s.mu.Unlock()
}

func (s *Service) requireEmptyStorage() error {
	_, metadataErr := s.metadata.Load()
	code, codeErr := s.codes.Load()
	clearBytes(code)
	metadataMissing := errors.Is(metadataErr, securestore.ErrNoRoomMetadata)
	codeMissing := errors.Is(codeErr, securestore.ErrNoProtectedRoomCode)
	if metadataErr == nil && codeErr == nil {
		return ErrRoomAlreadySaved
	}
	if metadataMissing && codeMissing {
		return nil
	}
	return ErrStoredStateConflict
}

func (s *Service) fail(err error) (Snapshot, error) {
	return s.machine.Apply(Facts{RecoverableError: true}), err
}

type enrollmentTransaction struct {
	service                *Service
	profileID              string
	enrollmentAttempted    bool
	codeWriteAttempted     bool
	metadataWriteAttempted bool
	compensated            bool
	cleanupFailures        int
}

func (t *enrollmentTransaction) compensate(parent context.Context) {
	if t.compensated {
		return
	}
	t.compensated = true
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), cleanupTimeout)
	defer cancel()
	if t.enrollmentAttempted {
		if err := t.service.netbird.Deregister(ctx, t.profileID); err != nil {
			t.cleanupFailures++
		}
	}
	if err := t.service.netbird.RemoveProfile(ctx, t.profileID); err != nil {
		t.cleanupFailures++
	}
	if t.metadataWriteAttempted {
		if err := t.service.metadata.Clear(); err != nil {
			t.cleanupFailures++
		}
	}
	if t.codeWriteAttempted {
		if err := t.service.codes.Clear(); err != nil {
			t.cleanupFailures++
		}
	}
}

func (t *enrollmentTransaction) wrap(cause error, parent context.Context) error {
	t.compensate(parent)
	return &TransactionError{Cause: cause, CleanupFailures: t.cleanupFailures}
}

func clearBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
