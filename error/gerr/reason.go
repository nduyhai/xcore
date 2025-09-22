package gerr

import (
	"github.com/nduyhai/xcore/error/xerr"
	"google.golang.org/grpc/codes"
)

type GRPCAware interface {
	GRPCCode() codes.Code
}

type GRPCReason struct {
	xerr.SimpleReason
	GrpcCode codes.Code `json:"grpc_code"`
}

func NewGRPCReason(code xerr.ErrorCode, message string, grpcCode codes.Code) xerr.Reason {
	return &GRPCReason{
		SimpleReason: xerr.SimpleReason{
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
	xerr.SimpleReason
	StatusCode int        `json:"http_status_code,omitempty"`
	GrpcCode   codes.Code `json:"grpc_code,omitempty"`
}

func NewMultiReason(code xerr.ErrorCode, message string, httpStatus int, grpcCode codes.Code) xerr.Reason {
	return &MultiReason{
		SimpleReason: xerr.SimpleReason{
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

func GetGRPCCode(reason xerr.Reason) codes.Code {
	if grpcReason, ok := reason.(GRPCAware); ok {
		return grpcReason.GRPCCode()
	}
	// Default fallback
	return codes.Unknown
}

func ErrorToGRPCCode(err xerr.Error) codes.Code {
	if err == nil {
		return codes.OK
	}
	return GetGRPCCode(err.Reason())
}
