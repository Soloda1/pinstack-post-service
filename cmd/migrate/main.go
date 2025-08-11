package main

import (
	"flag"
	"os"

	"pinstack-post-service/internal/infrastructure/config"
	"pinstack-post-service/internal/infrastructure/logger"
	"pinstack-post-service/internal/infrastructure/migrator"
)

func main() {
	cfg := config.MustLoad()

	log := logger.New(cfg.Env)

	command := flag.String("command", "up", "Migration command (up/down)")
	flag.Parse()

	dsn := "postgres://" + cfg.Database.Username + ":" + cfg.Database.Password + "@" +
		cfg.Database.Host + ":" + cfg.Database.Port + "/" + cfg.Database.DbName + "?sslmode=disable"

	m, err := migrator.NewMigrator(cfg.Database.MigrationsPath, dsn, log)
	if err != nil {
		log.Error("Failed to create migrator", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := m.Close(); err != nil {
			log.Error("Failed to close migrator", "error", err)
		}
	}()

	switch *command {
	case "up":
		if err := m.Up(); err != nil {
			log.Error("Failed to apply migrations", "error", err)
			os.Exit(1)
		}
		log.Info("Migrations applied successfully")
	case "down":
		if err := m.Down(); err != nil {
			log.Error("Failed to rollback migrations", "error", err)
			os.Exit(1)
		}
		log.Info("Migrations rolled back successfully")
	default:
		log.Error("Unknown command", "command", *command)
		os.Exit(1)
	}
}
