package gerr

import (
	"github.com/nduyhai/xcore/error/xerr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func FromGRPCStatus(st *status.Status) xerr.Error {
	if st == nil {
		return nil
	}

	var reason xerr.Reason

	// Look for ProtoReason in status details
	for _, detail := range st.Details() {
		if protoReason, ok := detail.(*ProtoReason); ok {
			reason = protoReason
			break
		}
		// Also check for any other xerr.Reason implementation
		if xeReason, ok := detail.(xerr.Reason); ok {
			reason = xeReason
			break
		}
	}

	// Default to ProtoReason if no custom reason found in details
	if reason == nil {
		reasonCode := xerr.ErrorCode(st.Code().String())
		message := st.Message()
		reason = NewProtoReasonWithCodes(reasonCode, message, nil, st.Code())
	}

	return xerr.New(reason, nil)
}

// ErrorToGRPCStatus converts xerr.Error to GRPC Status
func ErrorToGRPCStatus(err xerr.Error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}

	// Get GRPC code from reason
	grpcCode := xerr.GetGRPCCode(err.Reason())

	// Create status with basic info
	st := status.New(grpcCode, err.Error())

	// Convert reason to ProtoReason and add to details
	if err.Reason() != nil {
		var protoReason *ProtoReason

		// If the reason is already a ProtoReason, use it directly
		if existingProtoReason, ok := err.Reason().(*ProtoReason); ok {
			protoReason = existingProtoReason
		} else {
			// Convert other reason types to ProtoReason
			errorCode := err.Reason().Code()
			message := err.Reason().Message()

			// Check if reason has HTTP/GRPC codes
			var httpCode *int
			var gCode = xerr.GetGRPCCode(err.Reason())

			if httpAware, ok := err.Reason().(xerr.HTTPAware); ok {
				httpCodeVal := httpAware.HTTPCode()
				httpCode = &httpCodeVal
			}

			protoReason = NewProtoReasonWithCodes(errorCode, message, httpCode, gCode)
		}

		// Add ProtoReason to status details
		if detailedStatus, detailErr := st.WithDetails(protoReason); detailErr == nil {
			st = detailedStatus
		}
	}

	return st
}
