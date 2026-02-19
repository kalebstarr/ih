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
	"strings"
	"syscall"
	"time"

	_ "github.com/charmbracelet/bubbles"
	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/charmbracelet/harmonica"
	_ "github.com/charmbracelet/lipgloss"
	_ "github.com/lrstanley/bubblezone"
	"github.com/pressly/goose/v3"
	_ "github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

type Model struct {
	ctx context.Context

	db *sql.DB

	dbPath  string
	logPath string

	err error
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString("What should we buy at the market?\n\n")

	b.WriteString("\nPress q to quit.\n")
	return b.String()
}

func main() {
	var dbPathFlag string
	var logPathFlag string

	flag.StringVar(
		&dbPathFlag,
		"db",
		"",
		"Path to sqlite db file (default: OS config dir)",
	)
	flag.StringVar(
		&logPathFlag,
		"log",
		"",
		"Path to log file (default: OS config dir)",
	)
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

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("Failed to open config file: %v", err)
		os.Exit(1)
	}
	defer file.Close()
	handler := slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info(
		"starting",
		"os",
		runtime.GOOS,
		"arch",
		runtime.GOARCH,
		"db",
		dbPath,
		"log",
		logPath,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		slog.Error("open db failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()
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

			slog.Error("apply pragmas failed", "err", err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()

		slog.Error("ping db failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		slog.Error("set sql dialect failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	migCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := goose.UpContext(migCtx, db, "."); err != nil {
		slog.Error("migrations failed failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	m := Model{
		ctx:     ctx,
		db:      db,
		dbPath:  dbPath,
		logPath: logPath,
	}

	p := tea.NewProgram(m)

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
