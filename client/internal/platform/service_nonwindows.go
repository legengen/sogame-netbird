//go:build !windows

package platform

import "context"

type unsupportedServiceBackend struct{}

func newServiceBackend(string) ServiceBackend { return unsupportedServiceBackend{} }

func (unsupportedServiceBackend) Lookup(context.Context) (ServiceRecord, error) {
	return ServiceRecord{}, ErrServiceUnavailable
}
