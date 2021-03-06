package analogdb

import (
	"errors"
	"fmt"
)

const (
	ERRINTERNAL     = "internal"
	ERRNOTFOUND     = "not_found"
	ERRUNAUTHORIZED = "unauthorized"
)

type Error struct {
	Code    string
	Message string
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
