package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	mu     sync.Mutex
	file   *os.File
	logger *slog.Logger
}

func New(root string) (*Logger, error) {
	dir := filepath.Join(root, "log")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	path := filepath.Join(dir, time.Now().Format("2006-01-02")+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	handler := slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo})
	return &Logger{file: file, logger: slog.New(handler)}, nil
}

func (l *Logger) Info(message string, args ...any) {
	l.logger.Info(fmt.Sprintf(message, args...))
}

func (l *Logger) Error(message string, args ...any) {
	l.logger.Error(fmt.Sprintf(message, args...))
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}
