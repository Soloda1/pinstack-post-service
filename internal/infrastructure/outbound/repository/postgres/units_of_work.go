package postgres

import (
	"context"
	"fmt"
	ports "pinstack-post-service/internal/domain/ports/output"
	media_repository "pinstack-post-service/internal/domain/ports/output/media"
	post_repository "pinstack-post-service/internal/domain/ports/output/post"
	tag_repository "pinstack-post-service/internal/domain/ports/output/tag"
	media_repository_postgres "pinstack-post-service/internal/infrastructure/outbound/repository/media/postgres"
	post_repository_postgres "pinstack-post-service/internal/infrastructure/outbound/repository/post/postgres"
	tag_repository_postgres "pinstack-post-service/internal/infrastructure/outbound/repository/tag/postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate mockery --name UnitOfWork --dir . --output ../../../mocks --outpkg mocks --with-expecter --filename UnitsOfWork.go
type UnitOfWork interface {
	Begin(ctx context.Context) (Transaction, error)
}

//go:generate mockery --name Transaction --dir . --output ../../../mocks --outpkg mocks --with-expecter --filename Transaction.go
type Transaction interface {
	PostRepository() post_repository.Repository
	MediaRepository() media_repository.Repository
	TagRepository() tag_repository.Repository
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type PostgresUnitOfWork struct {
	pool *pgxpool.Pool
	log  ports.Logger
}

func NewPostgresUOW(pool *pgxpool.Pool, log ports.Logger) UnitOfWork {
	return &PostgresUnitOfWork{pool: pool, log: log}
}

func (uow *PostgresUnitOfWork) Begin(ctx context.Context) (Transaction, error) {
	tx, err := uow.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error beginning transaction: %w", err)
	}
	return &PostgresTransaction{tx: tx, log: uow.log}, nil
}

type PostgresTransaction struct {
	tx  pgx.Tx
	log ports.Logger
}

func (t *PostgresTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *PostgresTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (t *PostgresTransaction) PostRepository() post_repository.Repository {
	return post_repository_postgres.NewPostRepository(t.tx, t.log)
}

func (t *PostgresTransaction) MediaRepository() media_repository.Repository {
	return media_repository_postgres.NewMediaRepository(t.tx, t.log)
}

func (t *PostgresTransaction) TagRepository() tag_repository.Repository {
	return tag_repository_postgres.NewTagRepository(t.tx, t.log)
}
