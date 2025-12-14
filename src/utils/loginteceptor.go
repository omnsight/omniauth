package utils

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// LoggingInterceptor is a gRPC unary interceptor for logging.
func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {

	// 1. Get or Generate Request ID
	var requestID string

	// Check incoming metadata (headers) for an existing ID
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("x-request-id")
		if len(values) > 0 && values[0] != "" {
			requestID = values[0]
		}
	}

	// If no ID was provided, generate a new one
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// 2. Create a request-specific logger
	requestLogger := logrus.WithField("request_id", requestID)

	// 3. Add the logger to the context
	ctx = WithLogger(ctx, requestLogger)

	// Add a log entry for the start of the request
	requestLogger.WithFields(logrus.Fields{
		"method": info.FullMethod,
	}).Debug("gRPC request started")

	// 4. Call the original handler with the new context
	resp, err := handler(ctx, req)

	// 5. Log the end of the request
	if err != nil {
		requestLogger.WithError(err).Error("gRPC request finished with error")
	} else {
		requestLogger.Debug("gRPC request finished successfully")
	}

	return resp, err
}
