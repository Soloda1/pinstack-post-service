package user_client

import (
	"context"
	"log/slog"
	"pinstack-post-service/internal/custom_errors"
	"pinstack-post-service/internal/logger"
	"pinstack-post-service/internal/model"

	pb "github.com/soloda1/pinstack-proto-definitions/gen/go/pinstack-proto-definitions/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserClient struct {
	client pb.UserServiceClient
	log    *logger.Logger
}

func NewUserClient(conn *grpc.ClientConn, log *logger.Logger) *UserClient {
	return &UserClient{
		client: pb.NewUserServiceClient(conn),
		log:    log,
	}
}

func (u *UserClient) GetUser(ctx context.Context, id int64) (*model.User, error) {
	u.log.Info("Getting user by ID", slog.Int64("id", id))
	resp, err := u.client.GetUser(ctx, &pb.GetUserRequest{Id: id})
	if err != nil {
		u.log.Error("Error getting user", slog.String("error", err.Error()), slog.Int64("id", id))
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound {
				return nil, custom_errors.ErrUserNotFound
			}
		}
		return nil, custom_errors.ErrExternalServiceError
	}
	u.log.Info("Successfully got user", slog.Int64("id", id))
	return model.UserFromProto(resp), nil
}

func (u *UserClient) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	u.log.Info("Getting user by username", slog.String("username", username))
	resp, err := u.client.GetUserByUsername(ctx, &pb.GetUserByUsernameRequest{Username: username})
	if err != nil {
		u.log.Error("Failed to get user by username", slog.String("username", username), slog.String("error", err.Error()))
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound {
				return nil, custom_errors.ErrUserNotFound
			}
		}
		return nil, custom_errors.ErrExternalServiceError
	}
	u.log.Info("Successfully got user by username", slog.String("username", username))
	return model.UserFromProto(resp), nil
}

func (u *UserClient) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	u.log.Info("Getting user by email", slog.String("email", email))
	resp, err := u.client.GetUserByEmail(ctx, &pb.GetUserByEmailRequest{Email: email})
	if err != nil {
		u.log.Error("Failed to get user by email", slog.String("email", email), slog.String("error", err.Error()))
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound {
				return nil, custom_errors.ErrUserNotFound
			}
		}
		return nil, custom_errors.ErrExternalServiceError
	}
	u.log.Info("Successfully got user by email", slog.String("email", email))
	return model.UserFromProto(resp), nil
}
