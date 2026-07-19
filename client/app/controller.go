package app

import (
	"context"
	"log/slog"
	"sync"
)

const expectedNetBirdVersion = "0.74.7"

type Controller struct {
	mu     sync.RWMutex
	ctx    context.Context
	logger *slog.Logger
	state  StateSnapshot
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

func (c *Controller) Startup(ctx context.Context) {
	c.mu.Lock()
	c.ctx = ctx
	c.mu.Unlock()
	c.logger.Info("application started")
}

func (c *Controller) Shutdown(context.Context) {
	c.logger.Info("application stopped")
}

func (c *Controller) GetState() StateSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}
