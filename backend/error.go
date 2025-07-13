package analogdb

import (
	"errors"
	"fmt"
)

const (
	ERRINTERNAL      = "internal"
	ERRUNPROCESSABLE = "unprocessable"
	ERRNOTFOUND      = "not_found"
	ERRUNAVAILABLE   = "service_unavailable"
	ERRUNAUTHORIZED  = "unauthorized"
)

// Error represents an API error with code and message
type Error struct {
	Code    string `json:"code" example:"not_found"`
	Message string `json:"message" example:"not found"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("analogdb error: code: %s message: %s", e.Code, e.Message)
}

func ErrorMessage(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Message
	}
	return "Internal error"
}

func ErrorCode(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Code
	}
	return ERRINTERNAL
}
