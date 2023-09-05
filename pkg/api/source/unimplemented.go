package source

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HandleExternalRequestUnimplemented is used for plugins which doesn't implement HandleExternalRequest method.
type HandleExternalRequestUnimplemented struct{}

// HandleExternalRequest returns unimplemented error.
func (HandleExternalRequestUnimplemented) HandleExternalRequest(context.Context, ExternalRequestInput) (ExternalRequestOutput, error) {
	return ExternalRequestOutput{}, status.Errorf(codes.Unimplemented, "method HandleExternalRequest not implemented")
}

// StreamUnimplemented is used for plugins which doesn't implement Stream method.
type StreamUnimplemented struct{}

// Stream returns unimplemented error.
func (StreamUnimplemented) Stream(context.Context, StreamInput) (StreamOutput, error) {
	return StreamOutput{}, status.Errorf(codes.Unimplemented, "method Stream not implemented")
}
