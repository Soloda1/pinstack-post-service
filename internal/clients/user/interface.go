package user_client

import (
	"context"
	"pinstack-post-service/internal/model"
)

//go:generate mockery --name Client --dir . --output ../../../mocks --outpkg mocks --with-expecter --filename UserClient.go
type Client interface {
	GetUser(ctx context.Context, id int64) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
}
