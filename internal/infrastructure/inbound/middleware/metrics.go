package middleware

import (
	"context"
	"time"

	ports "pinstack-post-service/internal/domain/ports/output"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryMetricsInterceptor(metrics ports.MetricsProvider) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		start := time.Now()

		resp, err = handler(ctx, req)

		duration := time.Since(start)

		st := status.Code(err)
		statusStr := st.String()

		metrics.IncrementGRPCRequests(info.FullMethod, statusStr)
		metrics.RecordGRPCRequestDuration(info.FullMethod, statusStr, duration)

		return resp, err
	}
}
