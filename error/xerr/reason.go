package xerr

import (
	"net/http"

	"google.golang.org/grpc/codes"
)

type ErrorCode string

type Reason interface {
	Code() ErrorCode
	Message() string
}

type HTTPAware interface {
	HTTPCode() int
}

type GRPCAware interface {
	GRPCCode() codes.Code
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

type GRPCReason struct {
	SimpleReason
	GrpcCode codes.Code `json:"grpc_code"`
}

func NewGRPCReason(code ErrorCode, message string, grpcCode codes.Code) Reason {
	return &GRPCReason{
		SimpleReason: SimpleReason{
			ErrorCode:    code,
			ErrorMessage: message,
		},
		GrpcCode: grpcCode,
	}
}

func (r *GRPCReason) GRPCCode() codes.Code {
	return r.GrpcCode
}

type MultiReason struct {
	SimpleReason
	StatusCode int        `json:"http_status_code,omitempty"`
	GrpcCode   codes.Code `json:"grpc_code,omitempty"`
}

func NewMultiReason(code ErrorCode, message string, httpStatus int, grpcCode codes.Code) Reason {
	return &MultiReason{
		SimpleReason: SimpleReason{
			ErrorCode:    code,
			ErrorMessage: message,
		},
		StatusCode: httpStatus,
		GrpcCode:   grpcCode,
	}
}

func (r *MultiReason) HTTPCode() int {
	return r.StatusCode
}

func (r *MultiReason) GRPCCode() codes.Code {
	return r.GrpcCode
}

func GetHTTPCode(reason Reason) int {
	if httpReason, ok := reason.(HTTPAware); ok {
		return httpReason.HTTPCode()
	}
	// Default fallback
	return http.StatusInternalServerError
}

func GetGRPCCode(reason Reason) codes.Code {
	if grpcReason, ok := reason.(GRPCAware); ok {
		return grpcReason.GRPCCode()
	}
	// Default fallback
	return codes.Unknown
}

func ErrorToHTTPStatus(err Error) int {
	if err == nil {
		return http.StatusOK
	}
	return GetHTTPCode(err.Reason())
}

func ErrorToGRPCCode(err Error) codes.Code {
	if err == nil {
		return codes.OK
	}
	return GetGRPCCode(err.Reason())
}
