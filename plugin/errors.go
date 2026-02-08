package plugin

// ValidationError represents a user-facing validation error.
// When returned from plugin operations, the Message field contains
// a user-friendly error message that can be shown in the frontend.
type ValidationError struct {
	Err     error
	Message string // User-facing message
}

func (e *ValidationError) Error() string {
	if e == nil || e.Err == nil {
		return "validation error"
	}
	return e.Err.Error()
}

func (e *ValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewValidationError creates a ValidationError with the underlying error and user-facing message.
func NewValidationError(err error, message string) *ValidationError {
	return &ValidationError{Err: err, Message: message}
}
