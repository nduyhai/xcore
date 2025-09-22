package xerr

import (
	"net/http"
)

type ErrorCode string

type Reason interface {
	Code() ErrorCode
	Message() string
}

type HTTPAware interface {
	HTTPCode() int
}

type SimpleReason struct {
	ErrorCode    ErrorCode `json:"error_code"`
	ErrorMessage string    `json:"error_message"`
}

func NewSimpleReason(errorCode ErrorCode, message string) Reason {
	return &SimpleReason{ErrorCode: errorCode, ErrorMessage: message}
}

func (r *SimpleReason) Code() ErrorCode {
	return r.ErrorCode
}

func (r *SimpleReason) Message() string {
	return r.ErrorMessage
}

type HTTPReason struct {
	SimpleReason
	StatusCode int `json:"http_status_code"`
}

func NewHTTPReason(code ErrorCode, message string, httpStatus int) Reason {
	return &HTTPReason{
		SimpleReason: SimpleReason{
			ErrorCode:    code,
			ErrorMessage: message,
		},
		StatusCode: httpStatus,
	}
}

func (r *HTTPReason) HTTPCode() int {
	return r.StatusCode
}

func ErrorToHTTPStatus(err Error) int {
	if err == nil {
		return http.StatusOK
	}
	return GetHTTPCode(err.Reason())
}

func GetHTTPCode(reason Reason) int {
	if httpReason, ok := reason.(HTTPAware); ok {
		return httpReason.HTTPCode()
	}
	// Default fallback
	return http.StatusInternalServerError
}
