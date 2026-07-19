package platform

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	releasebuild "github.com/legengen/sogame-netbird/client/build"
)

type MSIAction string

const (
	MSIInstall MSIAction = "install"
	MSIRepair  MSIAction = "repair"
)

type ArtifactCheck interface {
	Verify(ctx context.Context, path string, expected releasebuild.WindowsArtifact) error
}

type MSIRunner interface {
	Run(ctx context.Context, action MSIAction, artifactPath, logPath string) error
}

type PrivilegedInstaller struct {
	artifacts ArtifactCheck
	runner    MSIRunner
}

func NewPrivilegedInstaller(artifacts ArtifactCheck, runner MSIRunner) *PrivilegedInstaller {
	return &PrivilegedInstaller{artifacts: artifacts, runner: runner}
}

func (i *PrivilegedInstaller) Execute(ctx context.Context, action MSIAction, artifactPath, logPath string, expected releasebuild.WindowsArtifact) error {
	if action != MSIInstall && action != MSIRepair {
		return ErrUnsupportedAction
	}
	if !filepath.IsAbs(artifactPath) || !strings.EqualFold(filepath.Ext(artifactPath), ".msi") {
		return fmt.Errorf("invalid NetBird MSI path")
	}
	if !filepath.IsAbs(logPath) || !strings.EqualFold(filepath.Ext(logPath), ".log") {
		return fmt.Errorf("invalid installer log path")
	}
	if err := i.artifacts.Verify(ctx, artifactPath, expected); err != nil {
		return fmt.Errorf("verify NetBird MSI before %s: %w", action, err)
	}
	if err := i.runner.Run(ctx, action, artifactPath, logPath); err != nil {
		return fmt.Errorf("run NetBird MSI %s: %w", action, err)
	}
	return nil
}

func BuildMSIArguments(action MSIAction, artifactPath, logPath string) ([]string, error) {
	if action != MSIInstall && action != MSIRepair {
		return nil, ErrUnsupportedAction
	}
	arguments := []string{
		"/i", artifactPath,
		"/quiet", "/qn", "/norestart",
		"/l*v", logPath,
		"AUTOSTART=0",
	}
	if action == MSIRepair {
		arguments = append(arguments, "REINSTALL=ALL", "REINSTALLMODE=vomus")
	}
	return arguments, nil
}
