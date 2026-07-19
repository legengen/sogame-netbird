//go:build !windows

package platform

func RequestInstallerElevation(string, MSIAction, string, string) error {
	return ErrServiceUnavailable
}

func RequestDaemonRemovalElevation(string, bool, string) error {
	return ErrServiceUnavailable
}
