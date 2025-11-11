package api

import "time"

type APIResponse[T any] struct {
	Data      T             `json:"data,omitempty"`
	Error     ErrorResponse `json:"error"`
	Status    int           `json:"status,omitempty"`
	Timestamp string        `json:"timestamp"`
	Version   string        `json:"version"`
}

type ErrorResponse struct {
	Message          string `json:"message"`
	DetailedResponse string `json:"details,omitempty"`
}

const (
	MsgMissingAuthHeader = "Missing authorization header"
	MsgInvalidAuthHeader = "Invalid authorization header format"
	MsgUnauthorized      = "Unauthorized"
	MsgInternalError     = "An internal error occurred"
	MsgAccessDenied      = "Access denied: token not authorized for this vault"
	MsgDisabledAPIKey    = "API key is disabled"
	MsgExpiredAPIKey     = "API key has expired"
	MsgAPIKeyNotFound    = "API key not found"
)

func NewErrorResponseWithMessage(message string) APIResponse[interface{}] {
	return APIResponse[interface{}]{
		Error: ErrorResponse{
			Message: message,
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0.0",
	}
}

func NewSuccessResponse[T any](code int, data T) APIResponse[T] {
	return APIResponse[T]{
		Status:    code,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0.0",
	}
}
