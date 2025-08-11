package logger

import (
	"log/slog"
	"os"
	ports "pinstack-post-service/internal/domain/ports/output"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

type Logger struct {
	*slog.Logger
}

func (l *Logger) With(args ...any) ports.Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}

func New(env string) *Logger {
	var log *slog.Logger
	switch env {
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		}))
	default:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		}))
	}

	return &Logger{log}
}
