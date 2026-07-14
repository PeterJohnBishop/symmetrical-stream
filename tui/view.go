package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m Model) View() tea.View {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Status: %s\n", m.status))

	b.WriteString("Activity Logs:\n")
	if len(m.logs) == 0 {
		b.WriteString("  (No activity yet...)\n")
	} else {
		for _, log := range m.logs {
			b.WriteString(fmt.Sprintf("  %s\n", log))
		}
	}

	b.WriteString("\n[Press 'q' or 'ctrl+c' to quit]\n")

	return tea.NewView(b.String())
}
