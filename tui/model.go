// Package tui handles the terminal presentation of the app
package tui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/peterjohnbishop/symmetrical-stream/signaling"
	"github.com/peterjohnbishop/symmetrical-stream/streaming"
)

type (
	connectedMsg    struct{}
	eventMsg        struct{ msg signaling.EventMessage }
	errorMsg        struct{ err error }
	Role            string
	webrtcStatusMsg struct{ status string }
)

const (
	RoleSender   Role = "Sender"
	RoleReceiver Role = "Receiver"
)

type Model struct {
	sm          *signaling.SignalingManager
	wm          *streaming.WebRTCManager
	status      string
	logs        []string
	role        Role
	pathInput   textinput.Model
	totpInput   textinput.Model
	totpDisplay string
	focusIndex  int
}

func InitialModel(sm *signaling.SignalingManager, wm *streaming.WebRTCManager) Model {
	pi := textinput.New()
	pi.Placeholder = "Enter file path to send..."
	pi.CharLimit = 256
	pi.SetWidth(40)

	ti := textinput.New()
	ti.Placeholder = "Enter 6-digit ID..."
	ti.CharLimit = 6
	ti.SetWidth(40)

	styles := ti.Styles()
	styles.Focused.Text = pinkStyle
	ti.SetStyles(styles)

	m := Model{
		sm:          sm,
		wm:          wm,
		status:      "Disconnected",
		logs:        []string{},
		role:        RoleSender,
		pathInput:   pi,
		totpInput:   ti,
		totpDisplay: "------",
		focusIndex:  0,
	}

	return updateFocus(m)
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.connectCmd(),
		waitForMessage(m.sm.MessageChan),
		waitForError(m.sm.ErrChan),
		waitForWebRTCStatus(m.wm.StatusChan),
	)
}

func (m Model) connectCmd() tea.Cmd {
	return func() tea.Msg {
		m.sm.ConnectToSignalingServer()
		if m.sm.Conn != nil {
			go m.sm.StartListening()
			m.sm.RegisterWithSignalingServer()
			return connectedMsg{}
		}
		return nil
	}
}

func waitForMessage(sub <-chan signaling.EventMessage) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-sub
		if !ok {
			return nil
		}
		return eventMsg{msg: msg}
	}
}

func waitForError(sub <-chan error) tea.Cmd {
	return func() tea.Msg {
		err, ok := <-sub
		if !ok {
			return nil
		}
		return errorMsg{err: err}
	}
}

func waitForWebRTCStatus(sub <-chan string) tea.Cmd {
	return func() tea.Msg {
		status, ok := <-sub
		if !ok {
			return nil
		}
		return webrtcStatusMsg{status: status}
	}
}
