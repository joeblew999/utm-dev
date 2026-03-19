// Package logging provides structured runtime logging for goup-util apps.
//
// It wraps log/slog to provide:
//   - Structured JSON log files (append-only, immutable — DuckDB-friendly)
//   - Human-readable console output
//   - Automatic platform detection (OS, arch, mobile vs desktop)
//   - Session tracking (each app run gets a unique session ID)
//
// Usage:
//
//	logger := logging.New(logging.Config{
//	    AppName: "my-app",
//	    LogDir:  "/path/to/logs",  // empty = os.TempDir()/goup-util-logs
//	})
//	defer logger.Close()
//
//	logger.Info("app started", "url", cfg.URL)
//	logger.Event("navigate", "url", "https://example.com", "tab", 0)
//	logger.Error("webview failed", "err", err)
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger is the runtime logger for goup-util apps.
type Logger struct {
	slog      *slog.Logger
	file      *os.File
	sessionID string
	appName   string
	mu        sync.Mutex
}

// Role identifies whether logging is from the dev tool or a user app.
type Role string

const (
	// RoleDev is goup-util itself (build, screenshot, install commands).
	RoleDev Role = "dev"
	// RoleApp is an application built with goup-util (webviewer, hybrid-dashboard).
	RoleApp Role = "app"
)

// Config configures the logger.
type Config struct {
	// AppName identifies the application in log entries.
	AppName string

	// Role distinguishes dev tool logs from user app logs.
	// Default: RoleApp.
	Role Role

	// LogDir is the directory for log files. If empty, uses
	// os.TempDir()/goup-util-logs.
	LogDir string

	// Console enables human-readable output to stderr.
	// Default: true.
	Console bool

	// Level sets the minimum log level. Default: slog.LevelInfo.
	Level slog.Level
}

// New creates a Logger that writes structured JSON to a log file.
// The log file is named <appname>-<date>.jsonl (JSON Lines format).
// Each line is a self-contained JSON object — perfect for DuckDB ingestion.
func New(cfg Config) (*Logger, error) {
	if cfg.AppName == "" {
		cfg.AppName = "goup-util"
	}
	if cfg.Role == "" {
		cfg.Role = RoleApp
	}

	logDir := cfg.LogDir
	if logDir == "" {
		logDir = filepath.Join(os.TempDir(), "goup-util-logs")
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("logging: create log dir: %w", err)
	}

	// One file per day, append-only (immutable log data)
	date := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s.jsonl", cfg.AppName, date))

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("logging: open log file: %w", err)
	}

	sessionID := fmt.Sprintf("%d", time.Now().UnixMilli())

	// Build writers: always file, optionally stderr
	var w io.Writer = f
	if cfg.Console {
		w = io.MultiWriter(f, os.Stderr)
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: cfg.Level,
	})

	// Detect platform using gioismobile for WASM, runtime.GOOS for native
	plat := DetectPlatform()

	// Add persistent fields to every log entry
	logger := slog.New(handler).With(
		"app", cfg.AppName,
		"role", string(cfg.Role),
		"session", sessionID,
		"os", plat.OS,
		"arch", plat.Arch,
		"mobile", plat.IsMobile,
	)

	l := &Logger{
		slog:      logger,
		file:      f,
		sessionID: sessionID,
		appName:   cfg.AppName,
	}

	l.Info("session started",
		"log_file", logPath,
		"pid", os.Getpid(),
	)

	return l, nil
}

// Close flushes and closes the log file.
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	l.Info("session ended")
	return l.file.Close()
}

// Info logs at INFO level.
func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

// Warn logs at WARN level.
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

// Error logs at ERROR level.
func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

// Debug logs at DEBUG level.
func (l *Logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

// Event logs a named event at INFO level with a "event" key.
// Use this for structured app lifecycle events:
//
//	logger.Event("navigate", "url", "https://...", "tab", 0)
//	logger.Event("screenshot", "output", "/tmp/shot.png", "width", 2400)
//	logger.Event("crash", "err", err, "stack", stack)
func (l *Logger) Event(name string, args ...any) {
	allArgs := make([]any, 0, len(args)+2)
	allArgs = append(allArgs, "event", name)
	allArgs = append(allArgs, args...)
	l.slog.Info("event", allArgs...)
}

// SessionID returns the unique session identifier for this app run.
func (l *Logger) SessionID() string {
	return l.sessionID
}

// LogPath returns the path to the current log file.
func (l *Logger) LogPath() string {
	if l.file == nil {
		return ""
	}
	return l.file.Name()
}
