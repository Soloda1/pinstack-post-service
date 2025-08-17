package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	post_service "pinstack-post-service/internal/application/service/post"
	"pinstack-post-service/internal/infrastructure/config"
	delivery_grpc "pinstack-post-service/internal/infrastructure/inbound/grpc"
	post_grpc "pinstack-post-service/internal/infrastructure/inbound/grpc/post"
	metrics_server "pinstack-post-service/internal/infrastructure/inbound/metrics"
	"pinstack-post-service/internal/infrastructure/logger"
	redis_cache "pinstack-post-service/internal/infrastructure/outbound/cache/redis"
	user_client "pinstack-post-service/internal/infrastructure/outbound/client/user"
	prometheus_metrics "pinstack-post-service/internal/infrastructure/outbound/metrics/prometheus"
	media_postgres "pinstack-post-service/internal/infrastructure/outbound/repository/media/postgres"
	post_postgres "pinstack-post-service/internal/infrastructure/outbound/repository/post/postgres"
	"pinstack-post-service/internal/infrastructure/outbound/repository/postgres"
	tag_postgres "pinstack-post-service/internal/infrastructure/outbound/repository/tag/postgres"
)

func main() {
	cfg := config.MustLoad()
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DbName)
	ctx := context.Background()
	log := logger.New(cfg.Env)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Error("Failed to parse postgres poolConfig", slog.String("error", err.Error()))
		os.Exit(1)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Error("Failed to create postgres pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	userServiceConn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", cfg.UserService.Address, cfg.UserService.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error("Failed to connect to user service", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func(userServiceConn *grpc.ClientConn) {
		err := userServiceConn.Close()
		if err != nil {
			log.Error("Failed to close user service connection", slog.String("error", err.Error()))
		}
	}(userServiceConn)

	userClient := user_client.NewUserClient(userServiceConn, log)

	log.Info("Connecting to Redis",
		slog.String("address", cfg.Redis.Address),
		slog.Int("port", cfg.Redis.Port),
		slog.Int("db", cfg.Redis.DB))
	redisClient, err := redis_cache.NewClient(cfg.Redis, log)
	if err != nil {
		log.Error("Failed to create Redis client", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("Failed to close Redis connection", slog.String("error", err.Error()))
		}
	}()

	metrics := prometheus_metrics.NewPrometheusMetricsProvider()

	metrics.SetServiceHealth(true)

	userCache := redis_cache.NewUserCache(redisClient, log, metrics)
	postCache := redis_cache.NewPostCache(redisClient, log, metrics)

	unitOfWork := postgres.NewPostgresUOW(pool, log, metrics)
	postRepo := post_postgres.NewPostRepository(pool, log, metrics)
	tagRepo := tag_postgres.NewTagRepository(pool, log)
	mediaRepo := media_postgres.NewMediaRepository(pool, log)

	originalPostService := post_service.NewPostService(postRepo, tagRepo, mediaRepo, unitOfWork, log, userClient, metrics)

	postService := post_service.NewPostServiceCacheDecorator(
		originalPostService,
		userCache,
		postCache,
		log,
		metrics,
	)

	postGRPCApi := post_grpc.NewPostGRPCService(postService, log)
	grpcServer := delivery_grpc.NewServer(postGRPCApi, cfg.GRPCServer.Address, cfg.GRPCServer.Port, log, metrics)

	metricsServer := metrics_server.NewMetricsServer(cfg.Prometheus.Address, cfg.Prometheus.Port, log)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	done := make(chan bool, 1)
	metricsDone := make(chan bool, 1)

	go func() {
		if err := grpcServer.Run(); err != nil {
			log.Error("gRPC server error", slog.String("error", err.Error()))
		}
		done <- true
	}()

	go func() {
		if err := metricsServer.Run(); err != nil {
			log.Error("Metrics server error", slog.String("error", err.Error()))
		}
		metricsDone <- true
	}()

	<-quit
	log.Info("Shutting down servers...")

	metrics.SetServiceHealth(false)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := grpcServer.Shutdown(); err != nil {
		log.Error("gRPC server shutdown error", slog.String("error", err.Error()))
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error("Metrics server shutdown error", slog.String("error", err.Error()))
	}

	<-done
	<-metricsDone

	log.Info("Server exited")
}
