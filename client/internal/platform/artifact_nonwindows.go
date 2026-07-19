//go:build !windows

package platform

import (
	"context"
	"fmt"

	releasebuild "github.com/legengen/sogame-netbird/client/build"
)

type WindowsSignatureVerifier struct{}

func (WindowsSignatureVerifier) Verify(context.Context, string, releasebuild.Publisher) error {
	return fmt.Errorf("%w: Authenticode verification requires Windows", ErrSignatureInvalid)
}
