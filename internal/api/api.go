package api

import "time"

type APIResponse[T any] struct {
	Data      T             `json:"data,omitempty"`
	Error     ErrorResponse `json:"error"`
	Status    int           `json:"status"`
	Timestamp string        `json:"timestamp"`
	Version   string        `json:"version"`
}

type ErrorResponse struct {
	Message          string `json:"message"`
	DetailedResponse string `json:"details,omitempty"`
}

func NewErrorResponse(code int, message string, details string) APIResponse[interface{}] {
	return APIResponse[interface{}]{
		Status: code,
		Error: ErrorResponse{
			Message:          message,
			DetailedResponse: details,
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0.0", //TODO get proper API versioning
	}
}

func NewSuccessResponse[T any](code int, data T) APIResponse[T] {
	return APIResponse[T]{
		Status:    code,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0.0", //TODO get proper API versioning

	}
}
