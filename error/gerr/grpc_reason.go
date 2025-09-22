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
