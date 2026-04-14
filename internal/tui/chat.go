package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/glamour"
)

var (
	titleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)

	botNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	userNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
)

type errMsg error

type ChatModel struct {
	viewport    viewport.Model
	messages    []string
	textarea    textarea.Model
	err         error
	Sending     bool
	Responder   func(string) (string, error)
}

func InitialChatModel(responder func(string) (string, error)) ChatModel {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 1000

	ta.SetWidth(80)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	vp := viewport.New(100, 20)
	vp.SetContent(`Welcome to Vectora Chat!
Type your question below and press Enter to send.
Ctrl+C to quit. /clear to reset history.`)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return ChatModel{
		textarea:    ta,
		viewport:    vp,
		messages:    []string{},
		Responder:   responder,
	}
}

func (m ChatModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.Sending {
				return m, nil
			}
			input := m.textarea.Value()
			if strings.TrimSpace(input) == "" {
				return m, nil
			}

			if input == "/clear" {
				m.messages = []string{}
				m.viewport.SetContent("History cleared.")
				m.textarea.Reset()
				return m, nil
			}

			m.messages = append(m.messages, userNameStyle.Render("You: ") + input)
			m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
			m.textarea.Reset()
			m.viewport.GotoBottom()

			m.Sending = true
			return m, func() tea.Msg {
				resp, err := m.Responder(input)
				if err != nil {
					return errMsg(err)
				}
				return resp
			}
		}

	case string:
		m.Sending = false
		
		// Render markdown with Glamour
		rendered, err := glamour.Render(msg, "dark")
		if err != nil {
			rendered = msg // Fallback to plain text
		}
		
		m.messages = append(m.messages, botNameStyle.Render("Vectora: ") + rendered)
		m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
		m.viewport.GotoBottom()
		return m, nil

	case errMsg:
		m.Sending = false
		m.err = msg
		m.messages = append(m.messages, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error: %v", msg)))
		m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
		m.viewport.GotoBottom()
		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - m.textarea.Height() - 2
		m.textarea.SetWidth(msg.Width)
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m ChatModel) View() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	header := titleStyle.Render("Vectora REPL") + " " + info
	
	footer := ""
	if m.Sending {
		footer = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("Thinking...")
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n%s",
		header,
		m.viewport.View(),
		m.textarea.View(),
		footer,
	)
}
