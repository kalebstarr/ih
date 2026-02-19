package main

import (
	"context"
	"database/sql"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
