//go:build windows

package platform

import (
	"context"
	"fmt"
	"os/exec"

	"golang.org/x/sys/windows"
)

type WindowsMSIRunner struct{}

func NewWindowsMSIRunner() WindowsMSIRunner { return WindowsMSIRunner{} }

func (WindowsMSIRunner) Run(ctx context.Context, action MSIAction, artifactPath, logPath string) error {
	if !windows.GetCurrentProcessToken().IsElevated() {
		return ErrElevationRequired
	}
	arguments, err := BuildMSIArguments(action, artifactPath, logPath)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, "msiexec.exe", arguments...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("msiexec failed: %w: %s", err, RedactInstallerOutput(string(output)))
	}
	return nil
}

func (WindowsMSIRunner) Remove(ctx context.Context, productCode, logPath string) error {
	if !windows.GetCurrentProcessToken().IsElevated() {
		return ErrElevationRequired
	}
	arguments, err := BuildMSIRemovalArguments(productCode, logPath)
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, "msiexec.exe", arguments...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("msiexec failed: %w: %s", err, RedactInstallerOutput(string(output)))
	}
	return nil
}

func RedactInstallerOutput(string) string {
	// MSI paths and logs are local metadata; never return upstream output across
	// the privilege boundary because it may contain machine identifiers.
	return "installer details retained in the local MSI log"
}
