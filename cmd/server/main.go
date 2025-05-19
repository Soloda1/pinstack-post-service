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

	"pinstack-post-service/config"
	user_client "pinstack-post-service/internal/clients/user"
	delivery_grpc "pinstack-post-service/internal/delivery/grpc"
	post_grpc "pinstack-post-service/internal/delivery/grpc/post"
	"pinstack-post-service/internal/logger"
	media_postgres "pinstack-post-service/internal/repository/media/postgres"
	post_postgres "pinstack-post-service/internal/repository/post/postgres"
	"pinstack-post-service/internal/repository/postgres"
	tag_postgres "pinstack-post-service/internal/repository/tag/postgres"
	post_service "pinstack-post-service/internal/service/post"
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
	defer userServiceConn.Close()

	userClient := user_client.NewUserClient(userServiceConn, log)
	unitOfWork := postgres.NewPostgresUOW(pool, log)
	postRepo := post_postgres.NewPostRepository(pool, log)
	tagRepo := tag_postgres.NewTagRepository(pool, log)
	mediaRepo := media_postgres.NewMediaRepository(pool, log)

	postService := post_service.NewPostService(postRepo, tagRepo, mediaRepo, unitOfWork, log, userClient)
	postGRPCApi := post_grpc.NewPostGRPCService(postService, log)
	grpcServer := delivery_grpc.NewServer(postGRPCApi, cfg.GRPCServer.Address, cfg.GRPCServer.Port, log)

	done := make(chan bool)
	go func() {
		if err := grpcServer.Run(); err != nil {
			log.Error("gRPC server error", slog.String("error", err.Error()))
		}
		done <- true
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down gRPC server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := grpcServer.Shutdown(); err != nil {
		log.Error("gRPC server shutdown error", slog.String("error", err.Error()))
	}
	<-done
	log.Info("Server exiting")
}
