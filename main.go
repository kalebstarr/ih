package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "github.com/charmbracelet/bubbles"
	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/charmbracelet/harmonica"
	_ "github.com/charmbracelet/lipgloss"
	_ "github.com/lrstanley/bubblezone"
	_ "github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

type Model struct {
	db       *sql.DB
	choices  []string
	cursor   int
	selected map[int]struct{}
}

func initialModel() (Model, error) {
	db, err := sql.Open("sqlite", "devuser:devpass@tcp(127.0.0.1:3306)/mydb")
	if err != nil {
		return Model{}, err
	}

	return Model{
		db:       db,
		choices:  []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},
		selected: make(map[int]struct{}),
	}, nil
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
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString("What should we buy at the market?\n\n")

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		fmt.Fprintf(&b, "%s [%s] %s\n", cursor, checked, choice)
	}

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

	init_model, err := initialModel()
	if err != nil {
		fmt.Printf("Failed to initialize: %v", err)
		os.Exit(1)
	}
	defer init_model.db.Close()

	p := tea.NewProgram(init_model)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
