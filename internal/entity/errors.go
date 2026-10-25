package entity

import (
	"errors"
	"fmt"
)

// ErrorKind classifies business errors independently of transport; the
// response package maps each kind to an HTTP status.
type ErrorKind int

const (
	KindNotFound ErrorKind = iota + 1
	KindValidation
	KindConflict
	KindUnauthorized
)

// Error is a business error whose Message is safe to show to clients. Any
// error that is not an *entity.Error is internal and is never exposed.
type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewNotFoundError(format string, args ...any) error {
	return &Error{Kind: KindNotFound, Message: fmt.Sprintf(format, args...)}
}

func NewValidationError(format string, args ...any) error {
	return &Error{Kind: KindValidation, Message: fmt.Sprintf(format, args...)}
}

func NewConflictError(format string, args ...any) error {
	return &Error{Kind: KindConflict, Message: fmt.Sprintf(format, args...)}
}

func NewUnauthorizedError(format string, args ...any) error {
	return &Error{Kind: KindUnauthorized, Message: fmt.Sprintf(format, args...)}
}

// IsKind reports whether err (or anything it wraps) is an entity error of kind k.
func IsKind(err error, k ErrorKind) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == k
}
