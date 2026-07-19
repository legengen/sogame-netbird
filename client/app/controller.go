package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	clientnetbird "github.com/legengen/sogame-netbird/client/internal/netbird"
	"github.com/legengen/sogame-netbird/client/internal/roomapi"
	"github.com/legengen/sogame-netbird/client/internal/session"
)

const expectedNetBirdVersion = "0.74.7"

type Controller struct {
	mu     sync.RWMutex
	ctx    context.Context
	logger *slog.Logger
	state  StateSnapshot
	rooms  RoomSession
	close  func() error
}

// RoomSession is the narrow command surface exposed from the session layer.
// The frontend never receives the underlying NetBird or Room API types.
type RoomSession interface {
	Create(context.Context, string) (session.Snapshot, error)
	Join(context.Context, string, string) (session.Snapshot, error)
}

func New(logger *slog.Logger) *Controller {
	return &Controller{
		logger: logger,
		state: StateSnapshot{
			State:         StateNoRoom,
			ConnectedPath: PathNone,
			Peers:         []PeerSnapshot{},
			Service: ServiceSnapshot{
				ExpectedVersion: expectedNetBirdVersion,
			},
		},
	}
}

func NewWithSession(logger *slog.Logger, rooms RoomSession, close func() error) *Controller {
	controller := New(logger)
	controller.rooms = rooms
	controller.close = close
	return controller
}

func (c *Controller) Startup(ctx context.Context) {
	c.mu.Lock()
	c.ctx = ctx
	c.mu.Unlock()
	c.logger.Info("application started")
}

func (c *Controller) Shutdown(context.Context) {
	if c.close != nil {
		if err := c.close(); err != nil {
			c.logger.Warn("close NetBird RPC adapter", "error", err)
		}
	}
	c.logger.Info("application stopped")
}

func (c *Controller) GetState() StateSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Controller) CreateRoom(request CreateRoomRequest) StateSnapshot {
	displayName := request.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName, _ = os.Hostname()
	}
	return c.runRoomCommand("create", func(ctx context.Context) (session.Snapshot, error) {
		if c.rooms == nil {
			return session.Snapshot{}, errors.New("room session is unavailable")
		}
		return c.rooms.Create(ctx, displayName)
	})
}

func (c *Controller) JoinRoom(request JoinRoomRequest) StateSnapshot {
	displayName := request.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName, _ = os.Hostname()
	}
	return c.runRoomCommand("join", func(ctx context.Context) (session.Snapshot, error) {
		if c.rooms == nil {
			return session.Snapshot{}, errors.New("room session is unavailable")
		}
		return c.rooms.Join(ctx, request.RoomCode, displayName)
	})
}

func (c *Controller) runRoomCommand(command string, execute func(context.Context) (session.Snapshot, error)) StateSnapshot {
	c.mu.Lock()
	if c.state.BusyCommand != "" {
		c.state.Error = &PublicError{
			Code:      ErrOperationConflict,
			Message:   "已有房间操作正在进行",
			Retryable: true,
			Action:    "等待当前操作完成",
		}
		state := c.state
		c.mu.Unlock()
		return state
	}
	c.state.BusyCommand = command
	c.state.Error = nil
	if c.state.State == StateNoRoom {
		c.state.State = StateEnrolling
	}
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Unlock()

	snapshot, err := execute(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.BusyCommand = ""
	if err != nil {
		c.state.State = StateRecoverableError
		c.state.ConnectedPath = PathNone
		c.state.Error = publicError(err)
		return c.state
	}
	c.state.Revision = snapshot.Revision
	c.state.State = ConnectionState(snapshot.State)
	c.state.ConnectedPath = PathType(snapshot.Path)
	c.state.Error = nil
	return c.state
}

func publicError(err error) *PublicError {
	if err == nil {
		return nil
	}
	var httpError *roomapi.HTTPError
	if errors.As(err, &httpError) {
		switch httpError.Code {
		case roomapi.ErrorRoomUnavailable:
			return &PublicError{Code: ErrRoomUnavailable, Message: "房间不存在或已不可用", Action: "检查房间码后重试"}
		case roomapi.ErrorRateLimited:
			return &PublicError{Code: ErrRoomAPIRateLimited, Message: "请求过于频繁", Retryable: true, Action: "稍后重试"}
		case roomapi.ErrorServiceUnavailable:
			return &PublicError{Code: ErrRoomAPIUnavailable, Message: "房间服务暂时不可用", Retryable: true, Action: "稍后重试"}
		}
		return &PublicError{Code: ErrInternal, Message: "房间请求未完成", Retryable: httpError.Transient(), Action: "稍后重试"}
	}
	if errors.Is(err, session.ErrRoomAlreadySaved) || errors.Is(err, session.ErrCommandInProgress) {
		return &PublicError{Code: ErrOperationConflict, Message: "当前已有一个已保存房间", Action: "先离开当前房间"}
	}
	if errors.Is(err, session.ErrStoredStateConflict) {
		return &PublicError{Code: ErrSecureStore, Message: "本地房间数据不完整", Action: "修复或离开当前房间"}
	}
	var mismatch *clientnetbird.VersionMismatchError
	if errors.As(err, &mismatch) {
		return &PublicError{Code: ErrVersionMismatch, Message: fmt.Sprintf("NetBird 版本不匹配，需要 v%s", mismatch.Expected), Action: "修复 NetBird 服务"}
	}
	if errors.Is(err, clientnetbird.ErrManagedProfileConflict) || errors.Is(err, clientnetbird.ErrManagedProfileInconsistent) {
		return &PublicError{Code: ErrProfileConflict, Message: "NetBird 管理配置与本地房间不一致", Action: "修复 NetBird 服务"}
	}
	if strings.Contains(strings.ToLower(err.Error()), "room session is unavailable") {
		return &PublicError{Code: ErrServiceUnavailable, Message: "NetBird 服务不可用", Retryable: true, Action: "检查服务后重试"}
	}
	return &PublicError{Code: ErrEnrollmentFailed, Message: "加入房间失败", Retryable: true, Action: "稍后重试"}
}
