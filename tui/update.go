package tui

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case connectedMsg:
		m.status = "Connected to Signaling Server"
		if m.sm.Identifier != "" {
			m.totpDisplay = m.sm.Identifier
		}
		return m, nil

	case webrtcStatusMsg:
		m.status = msg.status
		m.logs = append(m.logs, fmt.Sprintf("[WebRTC] %s", msg.status))
		return m, waitForWebRTCStatus(m.wm.StatusChan)

	case eventMsg:
		m.logs = append(m.logs, fmt.Sprintf("<- Received '%s' from %s", msg.msg.Type, msg.msg.Sender))

		// Route the incoming signaling event to the correct WebRTC phase
		switch msg.msg.Type {
		case "request_offer":
			if m.role == RoleSender {
				m.status = fmt.Sprintf("Receiver %s connecting. Generating Offer...", msg.msg.Sender)
				m.wm.StartWebRTC(true)
				if err := m.wm.SendOffer(msg.msg.Sender); err != nil {
					m.status = "Error: Failed to send offer"
					m.logs = append(m.logs, fmt.Sprintf("[ERROR] %v", err))
				}
			}

		case "offer":
			if m.role == RoleReceiver {
				m.status = "Offer received. Generating Answer..."
				m.wm.StartWebRTC(false)
				if err := m.wm.HandleOffer(msg.msg.Sender, msg.msg.Data); err != nil {
					m.status = "Error: Failed to handle offer"
					m.logs = append(m.logs, fmt.Sprintf("[ERROR] %v", err))
				}
			}

		case "answer":
			if m.role == RoleSender {
				m.status = "Answer received. Finalizing handshake..."
				if err := m.wm.HandleAnswer(msg.msg.Data); err != nil {
					m.status = "Error: Failed to handle answer"
					m.logs = append(m.logs, fmt.Sprintf("[ERROR] %v", err))
				}
			}

		case "candidate":
			if err := m.wm.HandleICECandidate(msg.msg.Data); err != nil {
				m.logs = append(m.logs, fmt.Sprintf("[WARN] Failed to apply ICE candidate: %v", err))
			}

		case "disconnect":
			m.status = fmt.Sprintf("Peer %s disconnected.", msg.msg.Sender)
			m.wm.DisconnectWebRTC()
		}

		return m, waitForMessage(m.sm.MessageChan)

	case errorMsg:
		m.status = "Error: check logs"
		m.logs = append(m.logs, fmt.Sprintf("[ERROR] %v", msg.err))
		return m, waitForError(m.sm.ErrChan)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.wm.DisconnectWebRTC()
			return m, tea.Quit

		case "enter":
			if m.role == RoleSender {
				path := m.pathInput.Value()
				if path == "" {
					m.status = "Error: Please enter a file path"
					return m, nil
				}

				if info, err := os.Stat(path); err != nil || info.IsDir() {
					m.status = "Error: Invalid file path"
					return m, nil
				}

				m.status = "File verified. Waiting for receiver to connect..."

			} else {
				dir := m.pathInput.Value()
				totp := m.totpInput.Value()

				if len(totp) != 6 {
					m.status = "Error: Sender ID must be exactly 6 digits"
					return m, nil
				}

				if info, err := os.Stat(dir); err != nil || !info.IsDir() {
					m.status = "Error: Invalid download directory"
					return m, nil
				}

				m.status = fmt.Sprintf("Pinging Sender %s to initiate WebRTC...", totp)

				// This kicks off the "request_offer" case on the Sender's side
				m.sm.SendEventMessage("request_offer", "Receiver ready", &totp)
			}
			return m, nil

		case "tab":
			m.focusIndex++
			if m.role == RoleSender && m.focusIndex > 1 {
				m.focusIndex = 0
			} else if m.role == RoleReceiver && m.focusIndex > 2 {
				m.focusIndex = 0
			}
			m = updateFocus(m)
			return m, nil

		case "shift+tab":
			m.focusIndex--
			if m.focusIndex < 0 {
				if m.role == RoleSender {
					m.focusIndex = 1
				} else {
					m.focusIndex = 2
				}
			}
			m = updateFocus(m)
			return m, nil

		case "space", "h", "l", "left", "right":
			if m.focusIndex == 0 {
				if m.role == RoleSender {
					m.role = RoleReceiver
					m.pathInput.Placeholder = "Enter download directory..."
				} else {
					m.role = RoleSender
					m.pathInput.Placeholder = "Enter file path to send..."
					if m.focusIndex == 2 {
						m.focusIndex = 1
					}
				}
				m = updateFocus(m)
				return m, nil
			}
		}

		switch m.focusIndex {
		case 1:
			var cmd tea.Cmd
			m.pathInput, cmd = m.pathInput.Update(msg)
			cmds = append(cmds, cmd)
		case 2:
			var cmd tea.Cmd
			m.totpInput, cmd = m.totpInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func updateFocus(m Model) Model {
	m.pathInput.Blur()
	m.totpInput.Blur()

	switch m.focusIndex {
	case 1:
		m.pathInput.Focus()
	case 2:
		m.totpInput.Focus()
	}
	return m
}
