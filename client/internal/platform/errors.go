package platform

import "errors"

var (
	ErrArtifactMissing   = errors.New("NetBird artifact is missing")
	ErrArtifactSize      = errors.New("NetBird artifact size mismatch")
	ErrArtifactDigest    = errors.New("NetBird artifact digest mismatch")
	ErrSignatureInvalid  = errors.New("NetBird artifact signature is invalid")
	ErrPublisherMismatch = errors.New("NetBird artifact publisher mismatch")
)
