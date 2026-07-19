//go:build !windows

package platform

func RequestInstallerElevation(string, MSIAction, string, string) error {
	return ErrServiceUnavailable
}
