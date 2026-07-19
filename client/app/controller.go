package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	clientnetbird "github.com/legengen/sogame-netbird/client/internal/netbird"
	"github.com/legengen/sogame-netbird/client/internal/platform"
	"github.com/legengen/sogame-netbird/client/internal/roomapi"
	"github.com/legengen/sogame-netbird/client/internal/session"
)

const expectedNetBirdVersion = "0.74.7"

type Controller struct {
	mu      sync.RWMutex
	ctx     context.Context
	logger  *slog.Logger
	state   StateSnapshot
	rooms   RoomSession
	close   func() error
	service ServiceChecker
	repair  func(context.Context) error
}

type ServiceChecker interface {
	Inspect(context.Context) (platform.ServiceInspection, error)
}

// RoomSession is the narrow command surface exposed from the session layer.
// The frontend never receives the underlying NetBird or Room API types.
type RoomSession interface {
	Create(context.Context, string) (session.Snapshot, error)
	Join(context.Context, string, string) (session.Snapshot, error)
}

type RoomViewSession interface {
	RoomSession
	View(context.Context) (session.RoomViewSnapshot, error)
	RevealRoomCode(context.Context) (string, error)
}

type RoomLifecycleSession interface {
	RoomViewSession
	Connect(context.Context) (session.Snapshot, error)
	Disconnect(context.Context) (session.Snapshot, error)
	Leave(context.Context) (session.Snapshot, error)
	Switch(context.Context, session.SwitchRequest) (session.Snapshot, error)
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

func (c *Controller) ConfigureService(checker ServiceChecker, repair func(context.Context) error) {
	c.mu.Lock()
	c.service = checker
	c.repair = repair
	c.mu.Unlock()
}

func (c *Controller) Startup(ctx context.Context) {
	c.mu.Lock()
	c.ctx = ctx
	c.mu.Unlock()
	c.logger.Info("application started")
	go c.refreshRoomView(ctx)
	go c.refreshService(ctx)
}

func (c *Controller) RepairService() StateSnapshot {
	c.mu.RLock()
	repair := c.repair
	ctx := c.ctx
	c.mu.RUnlock()
	if ctx == nil {
		ctx = context.Background()
	}
	if repair == nil {
		c.mu.Lock()
		c.state.Error = publicError(platform.ErrServiceUnavailable)
		c.state.Service.RepairRequired = true
		state := c.state
		c.mu.Unlock()
		return state
	}
	c.mu.Lock()
	c.state.BusyCommand = "repair"
	c.state.Error = nil
	c.mu.Unlock()
	err := repair(ctx)
	c.mu.Lock()
	c.state.BusyCommand = ""
	if err != nil {
		c.state.Error = publicError(err)
		c.state.Service.RepairRequired = true
		state := c.state
		c.mu.Unlock()
		return state
	}
	c.state.Error = nil
	c.state.Service.RepairRequired = false
	state := c.state
	c.mu.Unlock()
	c.refreshService(ctx)
	return state
}

func (c *Controller) refreshService(ctx context.Context) {
	c.mu.RLock()
	checker := c.service
	c.mu.RUnlock()
	if checker == nil {
		return
	}
	inspection, err := checker.Inspect(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Service = ServiceSnapshot{
		Installed:       inspection.Installed,
		Running:         inspection.Running,
		Version:         inspection.Version,
		ExpectedVersion: inspection.ExpectedVersion,
		RepairRequired:  inspection.Health != platform.ServiceReady,
	}
	if err != nil && c.state.Error == nil {
		c.state.Error = publicError(err)
	}
	if c.state.Error == nil {
		switch inspection.Health {
		case platform.ServiceMissing:
			c.state.Error = &PublicError{Code: ErrServiceMissing, Message: "NetBird 服务未安装", Retryable: true, Action: "修复 NetBird 服务"}
		case platform.ServiceVersionMismatch:
			c.state.Error = &PublicError{Code: ErrVersionMismatch, Message: "NetBird 服务版本不匹配", Action: "修复 NetBird 服务"}
		case platform.ServiceStopped, platform.ServiceUnhealthy:
			c.state.Error = &PublicError{Code: ErrServiceUnavailable, Message: "NetBird 服务未运行", Retryable: true, Action: "修复或启动服务"}
		}
	}
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

func (c *Controller) RevealRoomCode() RevealRoomCodeResult {
	c.mu.RLock()
	rooms := c.rooms
	ctx := c.ctx
	c.mu.RUnlock()
	view, ok := rooms.(RoomViewSession)
	if !ok {
		return RevealRoomCodeResult{Error: publicError(errors.New("room session is unavailable"))}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	roomCode, err := view.RevealRoomCode(ctx)
	if err != nil {
		return RevealRoomCodeResult{Error: publicError(err)}
	}
	return RevealRoomCodeResult{RoomCode: roomCode}
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

func (c *Controller) ConnectRoom() StateSnapshot {
	return c.runCommand("connect", "", func(ctx context.Context) (session.Snapshot, error) {
		lifecycle, ok := c.lifecycleSession()
		if !ok {
			return session.Snapshot{}, errors.New("room session is unavailable")
		}
		return lifecycle.Connect(ctx)
	})
}

func (c *Controller) DisconnectRoom() StateSnapshot {
	return c.runCommand("disconnect", "", func(ctx context.Context) (session.Snapshot, error) {
		lifecycle, ok := c.lifecycleSession()
		if !ok {
			return session.Snapshot{}, errors.New("room session is unavailable")
		}
		return lifecycle.Disconnect(ctx)
	})
}

func (c *Controller) LeaveRoom() StateSnapshot {
	return c.runCommand("leave", "", func(ctx context.Context) (session.Snapshot, error) {
		lifecycle, ok := c.lifecycleSession()
		if !ok {
			return session.Snapshot{}, errors.New("room session is unavailable")
		}
		return lifecycle.Leave(ctx)
	})
}

func (c *Controller) SwitchRoom(request SwitchRoomRequest) StateSnapshot {
	displayName := request.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName, _ = os.Hostname()
	}
	return c.runCommand("switch", "", func(ctx context.Context) (session.Snapshot, error) {
		lifecycle, ok := c.lifecycleSession()
		if !ok {
			return session.Snapshot{}, errors.New("room session is unavailable")
		}
		return lifecycle.Switch(ctx, session.SwitchRequest{
			Mode:      request.Mode,
			RoomCode:  request.RoomCode,
			Hostname:  displayName,
			Confirmed: request.Confirmed,
		})
	})
}

func (c *Controller) lifecycleSession() (RoomLifecycleSession, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	lifecycle, ok := c.rooms.(RoomLifecycleSession)
	return lifecycle, ok
}

func (c *Controller) runRoomCommand(command string, execute func(context.Context) (session.Snapshot, error)) StateSnapshot {
	return c.runCommand(command, StateEnrolling, execute)
}

func (c *Controller) runCommand(command string, initialState ConnectionState, execute func(context.Context) (session.Snapshot, error)) StateSnapshot {
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
	if initialState != "" && c.state.State == StateNoRoom {
		c.state.State = initialState
	}
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Unlock()

	snapshot, err := execute(ctx)
	if err == nil {
		c.refreshRoomView(ctx)
	}
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
	if snapshot.State == session.StateNoRoom {
		c.clearActiveRoom()
	}
	return c.state
}

func (c *Controller) clearActiveRoom() {
	c.state.RoomID = ""
	c.state.RoomCodeMasked = ""
	c.state.ManagementURL = ""
	c.state.LocalNetBirdIP = ""
	c.state.ProfileID = ""
	c.state.Peers = []PeerSnapshot{}
	c.state.PeersStale = false
	c.state.LastPeerRefreshAt = ""
}

func (c *Controller) refreshRoomView(ctx context.Context) {
	c.mu.RLock()
	rooms := c.rooms
	c.mu.RUnlock()
	viewSession, ok := rooms.(RoomViewSession)
	if !ok {
		return
	}
	view, err := viewSession.View(ctx)
	if err != nil {
		if errors.Is(err, session.ErrStoredStateConflict) || errors.Is(err, session.ErrRoomAlreadySaved) {
			c.mu.Lock()
			c.state.State = StateRecoverableError
			c.state.Error = publicError(err)
			c.mu.Unlock()
		}
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Revision = view.Session.Revision
	c.state.State = ConnectionState(view.Session.State)
	c.state.ConnectedPath = PathType(view.Session.Path)
	c.state.RoomID = view.Metadata.RoomID
	c.state.RoomCodeMasked = view.RoomCodeMasked
	c.state.ManagementURL = view.Metadata.ManagementURL
	c.state.ProfileID = view.Metadata.ProfileID
	c.state.LocalNetBirdIP = view.LocalNetBirdIP
	c.state.PeersStale = view.PeersStale
	if view.LastPeerRefresh.IsZero() {
		c.state.LastPeerRefreshAt = ""
	} else {
		c.state.LastPeerRefreshAt = view.LastPeerRefresh.UTC().Format(time.RFC3339)
	}
	c.state.Peers = make([]PeerSnapshot, 0, len(view.Peers))
	for _, peer := range view.Peers {
		path := PathNone
		if peer.Connected {
			path = PathType(view.Session.Path)
		}
		c.state.Peers = append(c.state.Peers, PeerSnapshot{
			ID:        peer.ID,
			Name:      peer.Name,
			NetBirdIP: peer.NetBirdIP,
			Connected: peer.Connected,
			Path:      path,
		})
	}
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
	if errors.Is(err, session.ErrSwitchConfirmationRequired) {
		return &PublicError{Code: ErrInvalidInput, Message: "切换房间需要确认先离开当前房间", Action: "勾选确认后重试"}
	}
	if errors.Is(err, session.ErrInvalidSwitchMode) {
		return &PublicError{Code: ErrInvalidInput, Message: "切换模式无效", Action: "选择创建或加入"}
	}
	var mismatch *clientnetbird.VersionMismatchError
	if errors.As(err, &mismatch) {
		return &PublicError{Code: ErrVersionMismatch, Message: fmt.Sprintf("NetBird 版本不匹配，需要 v%s", mismatch.Expected), Action: "修复 NetBird 服务"}
	}
	if errors.Is(err, clientnetbird.ErrManagedProfileConflict) || errors.Is(err, clientnetbird.ErrManagedProfileInconsistent) {
		return &PublicError{Code: ErrProfileConflict, Message: "NetBird 管理配置与本地房间不一致", Action: "修复 NetBird 服务"}
	}
	if errors.Is(err, platform.ErrServiceMissing) {
		return &PublicError{Code: ErrServiceMissing, Message: "NetBird 服务未安装", Retryable: true, Action: "修复 NetBird 服务"}
	}
	if errors.Is(err, platform.ErrServiceUnavailable) {
		return &PublicError{Code: ErrServiceUnavailable, Message: "NetBird 服务暂时不可用", Retryable: true, Action: "检查服务或修复"}
	}
	if errors.Is(err, platform.ErrServiceAccess) {
		return &PublicError{Code: ErrServiceUnavailable, Message: "无法读取 NetBird 服务状态", Retryable: true, Action: "重试或修复服务"}
	}
	if strings.Contains(strings.ToLower(err.Error()), "room session is unavailable") {
		return &PublicError{Code: ErrServiceUnavailable, Message: "NetBird 服务不可用", Retryable: true, Action: "检查服务后重试"}
	}
	return &PublicError{Code: ErrEnrollmentFailed, Message: "加入房间失败", Retryable: true, Action: "稍后重试"}
}
