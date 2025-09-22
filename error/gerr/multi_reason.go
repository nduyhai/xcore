package gerr

import (
	"github.com/nduyhai/xcore/error/xerr"
	"google.golang.org/grpc/codes"
)

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
