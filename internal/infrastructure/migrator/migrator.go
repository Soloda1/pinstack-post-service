package migrator

import (
	"errors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	ports "pinstack-post-service/internal/domain/ports/output"
)

type Migrator struct {
	m   *migrate.Migrate
	log ports.Logger
}

func NewMigrator(migrationsPath, dsn string, log ports.Logger) (*Migrator, error) {
	m, err := migrate.New(
		"file://"+migrationsPath,
		dsn,
	)
	if err != nil {
		return nil, err
	}

	return &Migrator{m: m, log: log}, nil
}

func (m *Migrator) Up() error {
	if err := m.m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		m.log.Error("Failed to apply migrations", "error", err)
		return err
	}
	return nil
}

func (m *Migrator) Down() error {
	if err := m.m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		m.log.Error("Failed to rollback migrations", "error", err)
		return err
	}
	return nil
}

func (m *Migrator) Close() error {
	sourceErr, dbErr := m.m.Close()
	if sourceErr != nil {
		m.log.Error("Failed to close source", "error", sourceErr)
	}
	if dbErr != nil {
		m.log.Error("Failed to close database connection", "error", dbErr)
	}
	return nil
}
