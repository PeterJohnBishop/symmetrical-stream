package tui

import (
	"fmt"
	"os"
	"time"

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

	case webrtcStatusMsg:
		m.status = msg.status
		m.logs = append(m.logs, fmt.Sprintf("[WebRTC] %s", msg.status))

		// SENDER TRIGGER: Data channel is open, start slicing and sending the file
		if msg.status == "Local Data channel is open. Sending..." && m.role == RoleSender {
			filePath := m.pathInput.Value()

			go func() {
				transmitFunc := func(data []byte) error {
					return m.wm.SafeWriteBytesToDC(data)
				}

				waitBufferFunc := func() {
					// Basic backpressure yield to prevent suffocating the Pion WebRTC routine
					m.wm.Mux.RLock()
					dc := m.wm.DC
					m.wm.Mux.RUnlock()

					if dc != nil {
						// Wait if more than 1MB is buffered in the data channel
						for dc.BufferedAmount() > 1024*1024 {
							time.Sleep(5 * time.Millisecond)
						}
					}
				}

				if err := m.cm.SendFile(filePath, transmitFunc, waitBufferFunc); err != nil {
					m.cm.ErrChan <- err
				}
			}()
		}

		// RECEIVER TRIGGER: Data channel is open, ensure output directory is locked in
		if msg.status == "Remote Data channel opened. Ready to receive..." && m.role == RoleReceiver {
			m.cm.SetOutDir(m.pathInput.Value())
		}

		return m, waitForWebRTCStatus(m.wm.StatusChan)

	case webrtcDataMsg:
		// Receiver intercepts incoming WebRTC bytes and routes them directly to the ChunkManager
		if m.role == RoleReceiver {
			m.cm.ProcessIncomingMessage(msg.data)
		}
		return m, waitForWebRTCData(m.wm.DataChan)

	case chunkStatusMsg:
		m.status = msg.status
		m.logs = append(m.logs, fmt.Sprintf("[Chunking] %s", msg.status))
		return m, waitForChunkStatus(m.cm.StatusChan)

	case chunkProgressMsg:
		m.progress = msg.progress
		return m, waitForChunkProgress(m.cm.ProgressChan)

	case chunkErrorMsg:
		m.status = "File Error: check logs"
		m.logs = append(m.logs, fmt.Sprintf("[File Error] %v", msg.err))
		return m, waitForChunkError(m.cm.ErrChan)

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
