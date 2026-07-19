package app

import (
	"encoding/json"

	"github.com/legengen/sogame-netbird/client/internal/observability"
)

type ErrorCode string

const (
	ErrInvalidInput       ErrorCode = "INVALID_INPUT"
	ErrRoomUnavailable    ErrorCode = "ROOM_UNAVAILABLE"
	ErrRoomAPIRateLimited ErrorCode = "ROOM_API_RATE_LIMITED"
	ErrRoomAPIUnavailable ErrorCode = "ROOM_API_UNAVAILABLE"
	ErrServiceMissing     ErrorCode = "NETBIRD_SERVICE_MISSING"
	ErrServiceUnavailable ErrorCode = "NETBIRD_SERVICE_UNAVAILABLE"
	ErrVersionMismatch    ErrorCode = "NETBIRD_VERSION_MISMATCH"
	ErrProfileConflict    ErrorCode = "NETBIRD_PROFILE_CONFLICT"
	ErrEnrollmentFailed   ErrorCode = "ENROLLMENT_FAILED"
	ErrOperationConflict  ErrorCode = "OPERATION_CONFLICT"
	ErrSecureStore        ErrorCode = "SECURE_STORE_UNAVAILABLE"
	ErrInternal           ErrorCode = "INTERNAL"
)

type PublicError struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Retryable bool      `json:"retryable"`
	Action    string    `json:"action,omitempty"`
}

func (e *PublicError) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Code)
}

func (e *PublicError) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	type publicErrorJSON PublicError
	clean := publicErrorJSON(*e)
	clean.Message = observability.Redact(clean.Message)
	clean.Action = observability.Redact(clean.Action)
	return json.Marshal(clean)
}
