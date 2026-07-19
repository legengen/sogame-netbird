//go:build !windows

package platform

import "context"

type WindowsMSIRunner struct{}

func NewWindowsMSIRunner() WindowsMSIRunner { return WindowsMSIRunner{} }

func (WindowsMSIRunner) Run(context.Context, MSIAction, string, string) error {
	return ErrServiceUnavailable
}
