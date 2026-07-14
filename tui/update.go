package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case connectedMsg:
		m.status = "🟢 Connected to Signaling Server"
		m.logs = append(m.logs, "WebSocket channel opened.")
		return m, nil

	case eventMsg:
		logLine := fmt.Sprintf("Received [%s] msg from %s", msg.msg.Type, msg.msg.Sender)
		m.logs = append(m.logs, logLine)

		if len(m.logs) > 10 {
			m.logs = m.logs[1:]
		}

		return m, waitForMessage(m.sm.MessageChan)

	case errorMsg:
		m.status = "🔴 Error"
		m.logs = append(m.logs, fmt.Sprintf("Error: %v", msg.err))

		return m, waitForError(m.sm.ErrChan)
	}

	return m, nil
}
