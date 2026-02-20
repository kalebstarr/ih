package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"ih/migrations"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/charmbracelet/harmonica"
	_ "github.com/charmbracelet/lipgloss"
	_ "github.com/lrstanley/bubblezone"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func setupLogger(logPath string) (*os.File, error) {
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	handler := slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
	return file, nil
}

func setupDB(ctx context.Context, dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}
	for _, p := range pragmas {
		if _, err := db.ExecContext(ctx, p); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func runMigrations(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	migCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return goose.UpContext(migCtx, db, ".")
}

func main() {
	var dbPathFlag string
	var logPathFlag string

	flag.StringVar(&dbPathFlag, "db", "", "Path to sqlite db file (default: OS config dir)")
	flag.StringVar(&logPathFlag, "log", "", "Path to log file (default: OS config dir)")
	flag.Parse()

	appName := "ih"
	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Printf("Failed to determine config directory: %v", err)
		os.Exit(1)
	}
	appDir := filepath.Join(configDir, appName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		fmt.Printf("Failed to create config dir: %v", err)
		os.Exit(1)
	}

	dbPath := dbPathFlag
	if dbPath == "" {
		dbPath = filepath.Join(appDir, "app.db")
	}
	logPath := logPathFlag
	if logPath == "" {
		logPath = filepath.Join(appDir, "app.log")
	}

	logFile, err := setupLogger(logPath)
	if err != nil {
		fmt.Printf("Failed to open log file: %v", err)
		os.Exit(1)
	}
	defer logFile.Close()

	slog.Info("starting",
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"db", dbPath,
		"log", logPath,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := setupDB(ctx, dbPath)
	if err != nil {
		slog.Error("setup db failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()

	if err := runMigrations(ctx, db); err != nil {
		slog.Error("migrations failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	m := Model{
		ctx:      ctx,
		db:       db,
		dbPath:   dbPath,
		logPath:  logPath,
		list: list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		<-ctx.Done()
		slog.Info("signal received, quitting TUI")
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
