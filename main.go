package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/charmbracelet/bubbles"
	_ "github.com/charmbracelet/lipgloss"
	_ "github.com/charmbracelet/harmonica"
	_ "github.com/lrstanley/bubblezone"
	_ "modernc.org/sqlite"
	_ "github.com/pressly/goose/v3"
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
