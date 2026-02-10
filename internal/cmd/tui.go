package cmd

import (
	"fmt"
	"python-comment-remover/internal/scanner"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	fileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	lineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
)

type TaskItem struct {
	File     string
	Comment  scanner.Comment
	Selected bool
}

type Model struct {
	items    []TaskItem
	cursor   int
	quitting bool
	done     bool
}

func NewModel(tasks map[string][]scanner.Comment) Model {
	var items []TaskItem
	for file, comments := range tasks {
		for _, comment := range comments {
			items = append(items, TaskItem{
				File:     file,
				Comment:  comment,
				Selected: true,
			})
		}
	}
	return Model{
		items:  items,
		cursor: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case " ", "x":
			if len(m.items) > 0 {
				m.items[m.cursor].Selected = !m.items[m.cursor].Selected
			}

		case "a":
			for i := range m.items {
				m.items[i].Selected = true
			}

		case "n":
			for i := range m.items {
				m.items[i].Selected = false
			}

		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return "Cancelled.\n"
	}

	if m.done {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("🗑️  Select comments to remove"))
	b.WriteString("\n\n")

	if len(m.items) == 0 {
		b.WriteString("No comments found.\n")
		return b.String()
	}

	for i, item := range m.items {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("▸ ")
		}

		checked := "[ ]"
		style := normalStyle
		if item.Selected {
			checked = "[✓]"
			style = selectedStyle
		}

		commentText := strings.TrimSpace(item.Comment.Text)
		if len(commentText) > 60 {
			commentText = commentText[:57] + "..."
		}
		commentText = strings.ReplaceAll(commentText, "\n", " ")

		line := fmt.Sprintf("%s %s %s:%s %s",
			cursor,
			checked,
			fileStyle.Render(item.File),
			lineStyle.Render(fmt.Sprintf("L%d", item.Comment.Line)),
			style.Render(commentText),
		)
		b.WriteString(line + "\n")
	}

	b.WriteString(helpStyle.Render("\n↑/↓: navigate • space/x: toggle • a: select all • n: deselect all • enter: confirm • q: quit"))

	return b.String()
}

func (m Model) GetSelectedTasks() map[string][]scanner.Comment {
	result := make(map[string][]scanner.Comment)
	for _, item := range m.items {
		if item.Selected {
			result[item.File] = append(result[item.File], item.Comment)
		}
	}
	return result
}

func (m Model) WasCancelled() bool {
	return m.quitting
}

func RunTUI(tasks map[string][]scanner.Comment) (map[string][]scanner.Comment, bool) {
	model := NewModel(tasks)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		return nil, true
	}

	m := finalModel.(Model)
	if m.WasCancelled() {
		return nil, true
	}

	return m.GetSelectedTasks(), false
}
