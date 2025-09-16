package xerr

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type Error interface {
	Error() string

	Reason() Reason

	StackTrace() StackTrace

	Metadata() map[string]string

	Cause() error

	WithMetadata(key string, value string) Error

	Unwrap() error

	Is(target error) bool

	Format(s fmt.State, verb rune)
}

type xError struct {
	reason     Reason
	stackTrace StackTrace
	metadata   map[string]string
	cause      error
}

func (e *xError) Error() string {
	if e.reason != nil {
		return e.reason.Message()
	}
	return "unknown error"
}

func (e *xError) Reason() Reason {
	return e.reason
}

func (e *xError) StackTrace() StackTrace {
	return e.stackTrace
}

func (e *xError) Metadata() map[string]string {
	if e.metadata == nil {
		return make(map[string]string)
	}
	result := make(map[string]string)
	for k, v := range e.metadata {
		result[k] = v
	}
	return result
}

func (e *xError) Cause() error {
	return e.cause
}

func (e *xError) WithMetadata(key string, value string) Error {
	if e.metadata == nil {
		e.metadata = make(map[string]string)
	}
	e.metadata[key] = value
	return e
}

func (e *xError) Unwrap() error {
	return e.cause
}

func (e *xError) Is(target error) bool {
	if target == nil {
		return false
	}
	if e == target {
		return true
	}
	var x *xError
	if errors.As(target, &x) {
		// Compare reasons instead of codes
		if e.reason != nil && x.reason != nil {
			return e.reason.Code() == x.reason.Code()
		}
		return e.reason == x.reason
	}
	if e.cause != nil {
		return e.cause == target
	}
	return false
}

func (e *xError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			str := strings.Join(e.StackTrace().Format(), "\n")
			_, _ = io.WriteString(s, str)
		}
	case 's':
		_, _ = fmt.Fprintf(s, "%s", e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	}
}

func New(reason Reason, causeErr error) Error {
	return &xError{
		reason:     reason,
		cause:      causeErr,
		stackTrace: NewStackTrace(),
		metadata:   make(map[string]string),
	}
}

func Wrap(err error, reason Reason) Error {
	if err == nil {
		return &xError{
			reason:     reason,
			stackTrace: NewStackTrace(),
			metadata:   make(map[string]string),
		}
	}

	var se *xError
	if errors.As(err, &se) {
		se.reason = reason
		return se
	}

	return &xError{
		reason:     reason,
		cause:      err,
		stackTrace: NewStackTrace(),
		metadata:   make(map[string]string),
	}
}
