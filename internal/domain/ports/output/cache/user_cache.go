package cache

import (
	"context"
	model "pinstack-post-service/internal/domain/models"
)

//go:generate mockery --name UserCache --dir . --output ../../../../mocks/cache --outpkg mocks --with-expecter --filename UserCache.go
type UserCache interface {
	GetUser(ctx context.Context, userID int64) (*model.User, error)
	SetUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, userID int64) error
}
