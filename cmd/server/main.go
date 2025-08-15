package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	post_service "pinstack-post-service/internal/application/service/post"
	"pinstack-post-service/internal/infrastructure/config"
	delivery_grpc "pinstack-post-service/internal/infrastructure/inbound/grpc"
	post_grpc "pinstack-post-service/internal/infrastructure/inbound/grpc/post"
	"pinstack-post-service/internal/infrastructure/logger"
	redis_cache "pinstack-post-service/internal/infrastructure/outbound/cache/redis"
	user_client "pinstack-post-service/internal/infrastructure/outbound/client/user"
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

	userCache := redis_cache.NewUserCache(redisClient, log)
	postCache := redis_cache.NewPostCache(redisClient, log)
	tagCache := redis_cache.NewTagCache(redisClient, log)

	unitOfWork := postgres.NewPostgresUOW(pool, log)
	postRepo := post_postgres.NewPostRepository(pool, log)
	tagRepo := tag_postgres.NewTagRepository(pool, log)
	mediaRepo := media_postgres.NewMediaRepository(pool, log)

	originalPostService := post_service.NewPostService(postRepo, tagRepo, mediaRepo, unitOfWork, log, userClient)

	postService := post_service.NewPostServiceCacheDecorator(
		originalPostService,
		userCache,
		postCache,
		tagCache,
		log,
	)

	postGRPCApi := post_grpc.NewPostGRPCService(postService, log)
	grpcServer := delivery_grpc.NewServer(postGRPCApi, cfg.GRPCServer.Address, cfg.GRPCServer.Port, log)

	metricsAddr := fmt.Sprintf("%s:%d", cfg.Prometheus.Address, cfg.Prometheus.Port)
	metricsServer := &http.Server{
		Addr:    metricsAddr,
		Handler: nil,
	}

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

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Info("Starting Prometheus metrics server", slog.String("address", metricsAddr))
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Prometheus metrics server error", slog.String("error", err.Error()))
		}
		metricsDone <- true
	}()

	<-quit
	log.Info("Shutting down servers...")

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
