package middleware

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"log/slog"
	"pinstack-post-service/internal/logger"
)

func UnaryLoggerInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		start := time.Now()

		var remoteAddr string
		if p, ok := peer.FromContext(ctx); ok {
			remoteAddr = p.Addr.String()
		}

		resp, err = handler(ctx, req)

		latency := time.Since(start)
		st, _ := status.FromError(err)

		log.With(
			slog.String("method", info.FullMethod),
			slog.String("remote_address", remoteAddr),
			slog.String("latency", latency.String()),
			slog.String("grpc_code", st.Code().String()),
		).Info("gRPC request completed")

		return resp, err
	}
}
