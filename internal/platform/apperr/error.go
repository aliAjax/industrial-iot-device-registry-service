package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

type Kind string

const (
	KindInvalid      Kind = "invalid"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindUnavailable  Kind = "unavailable"
	KindInternal     Kind = "internal"
	KindRateLimited  Kind = "rate_limited"
	KindBackpressure Kind = "backpressure"
)

type Error struct {
	Kind    Kind
	Op      string
	Message string
	Cause   error
	TraceID string
}

func (e *Error) Error() string {
	msg := e.Message
	if msg == "" {
		msg = string(e.Kind)
	}
	if e.Op != "" {
		msg = e.Op + ": " + msg
	}
	if e.Cause != nil {
		msg = msg + ": " + e.Cause.Error()
	}
	return msg
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func E(kind Kind, op, message string, cause error) error {
	return &Error{Kind: kind, Op: op, Message: message, Cause: cause}
}

func WithTraceID(err error, traceID string) error {
	var target *Error
	if errors.As(err, &target) {
		clone := *target
		clone.TraceID = traceID
		return &clone
	}
	return &Error{Kind: KindInternal, Message: err.Error(), TraceID: traceID}
}

func KindOf(err error) Kind {
	var target *Error
	if errors.As(err, &target) {
		return target.Kind
	}
	return KindInternal
}

func StatusCode(err error) int {
	switch KindOf(err) {
	case KindInvalid:
		return http.StatusBadRequest
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindUnavailable:
		return http.StatusServiceUnavailable
	case KindRateLimited:
		return http.StatusTooManyRequests
	case KindBackpressure:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func IsKind(err error, kind Kind) bool {
	return KindOf(err) == kind
}

func Errorf(kind Kind, op string, format string, args ...any) error {
	return &Error{Kind: kind, Op: op, Message: fmt.Sprintf(format, args...)}
}
