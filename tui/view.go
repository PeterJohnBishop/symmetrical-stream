package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	pinkStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5FD7")).Bold(true) // Pink
	purpleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#AF5FFF")).Bold(true) // Purple
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	pointer      = purpleStyle.Render("> ")
	emptyPointer = "  "

	logo = `     ████ █   █ █   █ █   █ █████ █████ ████  ███  ███   ███  █           ████ █████ ████  █████  ███  █   █ 
   █      █ █  ██ ██ ██ ██ █       █   █   █  █  █     █   █ █          █       █   █   █ █     █   █ ██ ██  
   ███    █   █ █ █ █ █ █ ████    █   ████   █  █     █████ █     ████  ███    █   ████  ████  █████ █ █ █   
     █   █   █   █ █   █ █       █   █  █   █  █     █   █ █              █   █   █  █  █     █   █ █   █    
████    █   █   █ █   █ █████   █   █   █ ███  ███  █   █ █████      ████    █   █   █ █████ █   █ █   █     `
)

func (m Model) View() tea.View {
	var s strings.Builder
	s.WriteString("\n\n")
	s.WriteString(pinkStyle.Render(logo))
	s.WriteString("\n\n")
	s.WriteString("	- file transfer over webrtc\n\n")
	s.WriteString(fmt.Sprintf("Status: %s\n\n", m.status))

	if m.progress > 0 && m.progress <= 100 {
		bar := ""
		for i := 0; i < 20; i++ {
			if float64(i) < (float64(m.progress)/100.0)*20 {
				bar += pinkStyle.Render("█")
			} else {
				bar += dimStyle.Render("░")
			}
		}
		s.WriteString(fmt.Sprintf("Progress: [%s] %d%%\n\n", bar, m.progress))
	}

	// role selection
	rolePtr := emptyPointer
	if m.focusIndex == 0 {
		rolePtr = pointer
	}

	senderText, receiverText := "Sender", "Receiver"
	if m.role == RoleSender {
		senderText = pinkStyle.Render(senderText)
		receiverText = dimStyle.Render(receiverText)
	} else {
		senderText = dimStyle.Render(senderText)
		receiverText = pinkStyle.Render(receiverText)
	}
	s.WriteString(fmt.Sprintf("%s%s   %s\n\n", rolePtr, senderText, receiverText))

	// directory / file path input
	pathPtr := emptyPointer
	if m.focusIndex == 1 {
		pathPtr = pointer
	}

	if m.role == RoleSender {
		s.WriteString(fmt.Sprintf("%sFile Path: %s\n", pathPtr, m.pathInput.View()))

		// code display
		totpPtr := emptyPointer // Sender cannot focus this line
		highlightedTOTP := pinkStyle.Render(m.totpDisplay)
		s.WriteString(fmt.Sprintf("%sCode: %s\n", totpPtr, highlightedTOTP))
	} else {
		s.WriteString(fmt.Sprintf("%sDownload Dir: %s\n", pathPtr, m.pathInput.View()))

		// code input
		totpPtr := emptyPointer
		if m.focusIndex == 2 {
			totpPtr = pointer
		}
		s.WriteString(fmt.Sprintf("%sCode Verification: %s\n", totpPtr, m.totpInput.View()))
	}

	// activity
	if len(m.logs) > 0 {
		s.WriteString("\nLogs:\n")
		for _, log := range m.logs {
			s.WriteString(fmt.Sprintf("- %s\n", log))
		}
	}

	s.WriteString("\n[tab/shift+tab: focus] • [h/l/space: toggle role] • [ctrl+c: quit]\n")

	return tea.NewView(s.String())
}
