package domain

import (
	"errors"
	"fmt"
)

type Kind string

const (
	KindInvalid       Kind = "invalid"
	KindNotFound      Kind = "not_found"
	KindConflict      Kind = "conflict"
	KindUnauthorized  Kind = "unauthorized"
	KindForbidden     Kind = "forbidden"
	KindNotAttendable Kind = "not_attendable"
	KindUnprocessable Kind = "unprocessable"
	KindRateLimited   Kind = "rate_limited"
)

type Detail struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type Error struct {
	Kind    Kind
	Message string
	Details []Detail
}

func (e *Error) Error() string { return e.Message }

func NewError(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}

func Errorf(kind Kind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

func NewErrorWithDetails(kind Kind, message string, details []Detail) *Error {
	return &Error{Kind: kind, Message: message, Details: details}
}

func ErrNotFound(what string) *Error {
	return NewError(KindNotFound, what+" not found")
}

func KindOf(err error) (Kind, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind, true
	}
	return "", false
}
