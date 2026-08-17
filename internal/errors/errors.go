package errors

import (
	"fmt"
	"net/http"
)

type ErrorCode string

const (
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	CodeForbidden        ErrorCode = "FORBIDDEN"
	CodeConflict         ErrorCode = "CONFLICT"
	CodeInvalidInput     ErrorCode = "INVALID_INPUT"
	CodeInternal         ErrorCode = "INTERNAL"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	CodeTooManyRequests  ErrorCode = "TOO_MANY_REQUESTS"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeConflict:
		return http.StatusConflict
	case CodeInvalidInput:
		return http.StatusBadRequest
	case CodeTooManyRequests:
		return http.StatusTooManyRequests
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func (c ErrorCode) String() string {
	return string(c)
}

func NewNotFound(msg string, err error) *AppError {
	return &AppError{Code: CodeNotFound, Message: msg, Err: err}
}

func NewUnauthorized(msg string, err error) *AppError {
	return &AppError{Code: CodeUnauthorized, Message: msg, Err: err}
}

func NewForbidden(msg string, err error) *AppError {
	return &AppError{Code: CodeForbidden, Message: msg, Err: err}
}

func NewConflict(msg string, err error) *AppError {
	return &AppError{Code: CodeConflict, Message: msg, Err: err}
}

func NewInvalidInput(msg string, err error) *AppError {
	return &AppError{Code: CodeInvalidInput, Message: msg, Err: err}
}

func NewInternal(msg string, err error) *AppError {
	return &AppError{Code: CodeInternal, Message: msg, Err: err}
}

func NewServiceUnavailable(msg string, err error) *AppError {
	return &AppError{Code: CodeServiceUnavailable, Message: msg, Err: err}
}

func NewTooManyRequests(msg string, err error) *AppError {
	return &AppError{Code: CodeTooManyRequests, Message: msg, Err: err}
}

func IsNotFound(err error) bool {
	var appErr *AppError
	return errorsAs(err, &appErr) && appErr.Code == CodeNotFound
}

func IsUnauthorized(err error) bool {
	var appErr *AppError
	return errorsAs(err, &appErr) && appErr.Code == CodeUnauthorized
}

func IsConflict(err error) bool {
	var appErr *AppError
	return errorsAs(err, &appErr) && appErr.Code == CodeConflict
}

func IsInvalidInput(err error) bool {
	var appErr *AppError
	return errorsAs(err, &appErr) && appErr.Code == CodeInvalidInput
}

func errorsAs(err error, target interface{}) bool {
	var appErr *AppError
	for e := err; e != nil; e = unwrap(e) {
		if e, ok := e.(*AppError); ok {
			appErr = e
			break
		}
	}
	if appErr == nil {
		return false
	}
	switch t := target.(type) {
	case **AppError:
		*t = appErr
		return true
	}
	return false
}

func unwrap(err error) error {
	type unwrapper interface {
		Unwrap() error
	}
	if u, ok := err.(unwrapper); ok {
		return u.Unwrap()
	}
	return nil
}