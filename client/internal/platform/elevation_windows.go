//go:build windows

package platform

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func RequestInstallerElevation(helperPath string, action MSIAction, artifactPath, logPath string) error {
	if action != MSIInstall && action != MSIRepair {
		return ErrUnsupportedAction
	}
	if !filepath.IsAbs(helperPath) || !strings.EqualFold(filepath.Base(helperPath), "sogame-helper.exe") {
		return fmt.Errorf("invalid elevated helper path")
	}
	if !filepath.IsAbs(artifactPath) || !filepath.IsAbs(logPath) {
		return fmt.Errorf("elevated helper requires absolute paths")
	}
	arguments := []string{
		"--action", string(action),
		"--artifact", artifactPath,
		"--log", logPath,
	}
	escaped := make([]string, len(arguments))
	for index, argument := range arguments {
		escaped[index] = syscall.EscapeArg(argument)
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(helperPath)
	parameters, _ := windows.UTF16PtrFromString(strings.Join(escaped, " "))
	directory, _ := windows.UTF16PtrFromString(filepath.Dir(helperPath))
	if err := windows.ShellExecute(0, verb, file, parameters, directory, windows.SW_HIDE); err != nil {
		return fmt.Errorf("request NetBird installer elevation: %w", err)
	}
	return nil
}

func RequestDaemonRemovalElevation(helperPath string, confirmed bool, logPath string) error {
	if !confirmed {
		return ErrRemovalNotConfirmed
	}
	if !filepath.IsAbs(helperPath) || !strings.EqualFold(filepath.Base(helperPath), "sogame-helper.exe") {
		return fmt.Errorf("invalid elevated helper path")
	}
	if !filepath.IsAbs(logPath) {
		return fmt.Errorf("elevated helper requires an absolute log path")
	}
	arguments := []string{"--action", string(MSIRemove), "--log", logPath}
	escaped := make([]string, len(arguments))
	for index, argument := range arguments {
		escaped[index] = syscall.EscapeArg(argument)
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(helperPath)
	parameters, _ := windows.UTF16PtrFromString(strings.Join(escaped, " "))
	directory, _ := windows.UTF16PtrFromString(filepath.Dir(helperPath))
	if err := windows.ShellExecute(0, verb, file, parameters, directory, windows.SW_HIDE); err != nil {
		return fmt.Errorf("request NetBird removal elevation: %w", err)
	}
	return nil
}
