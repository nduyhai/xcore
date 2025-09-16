package gerr

import (
	"net/http"

	"github.com/nduyhai/xcore/error/xerr"
	"google.golang.org/grpc/codes"
)

// Ensure ProtoReason implements xerr.Reason interface
var _ xerr.Reason = (*ProtoReason)(nil)

// Code implements xerr.Reason interface
func (p *ProtoReason) Code() xerr.ErrorCode {
	return xerr.ErrorCode(p.ErrorCode)
}

// Message implements xerr.Reason interface
func (p *ProtoReason) Message() string {
	return p.ErrorMessage
}

// HTTPCode implements xerr.HTTPAware interface
func (p *ProtoReason) HTTPCode() int {
	if p.HttpCode != nil {
		return int(*p.HttpCode)
	}
	// Default fallback
	return http.StatusInternalServerError
}

// GRPCCode implements xerr.GRPCAware interface
func (p *ProtoReason) GRPCCode() codes.Code {
	if p.GrpcCode != nil {
		return codes.Code(*p.GrpcCode)
	}
	// Default fallback
	return codes.Unknown
}

// NewProtoReason creates a new ProtoReason with basic fields
func NewProtoReason(errorCode xerr.ErrorCode, message string) *ProtoReason {
	return &ProtoReason{
		ErrorCode:    string(errorCode),
		ErrorMessage: message,
	}
}

// NewProtoReasonWithHTTP creates a new ProtoReason with HTTP code
func NewProtoReasonWithHTTP(errorCode xerr.ErrorCode, message string, httpCode int) *ProtoReason {
	httpCode32 := int32(httpCode)
	return &ProtoReason{
		ErrorCode:    string(errorCode),
		ErrorMessage: message,
		HttpCode:     &httpCode32,
	}
}

// NewProtoReasonWithGRPC creates a new ProtoReason with GRPC code
func NewProtoReasonWithGRPC(errorCode xerr.ErrorCode, message string, grpcCode codes.Code) *ProtoReason {
	grpcCode32 := int32(grpcCode)
	return &ProtoReason{
		ErrorCode:    string(errorCode),
		ErrorMessage: message,
		GrpcCode:     &grpcCode32,
	}
}

// NewProtoReasonWithCodes creates a new ProtoReason with both HTTP and GRPC codes
func NewProtoReasonWithCodes(errorCode xerr.ErrorCode, message string, httpCode *int, grpcCode codes.Code) *ProtoReason {
	reason := &ProtoReason{
		ErrorCode:    string(errorCode),
		ErrorMessage: message,
	}

	if httpCode != nil {
		httpCode32 := int32(*httpCode)
		reason.HttpCode = &httpCode32
	}

	grpcCode32 := int32(grpcCode)
	reason.GrpcCode = &grpcCode32

	return reason
}
