// Package tui handles the terminal presentation of the app
package tui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/peterjohnbishop/symmetrical-stream/chunking"
	"github.com/peterjohnbishop/symmetrical-stream/signaling"
	"github.com/peterjohnbishop/symmetrical-stream/streaming"
)

type (
	connectedMsg     struct{}
	eventMsg         struct{ msg signaling.EventMessage }
	errorMsg         struct{ err error }
	Role             string
	webrtcStatusMsg  struct{ status string }
	webrtcDataMsg    struct{ data []byte }
	chunkStatusMsg   struct{ status string }
	chunkProgressMsg struct{ progress int }
	chunkErrorMsg    struct{ err error }
)

const (
	RoleSender   Role = "Sender"
	RoleReceiver Role = "Receiver"
)

type Model struct {
	sm          *signaling.SignalingManager
	wm          *streaming.WebRTCManager
	cm          *chunking.ChunkManager
	status      string
	logs        []string
	role        Role
	pathInput   textinput.Model
	totpInput   textinput.Model
	totpDisplay string
	focusIndex  int
	progress    int
}

func InitialModel(sm *signaling.SignalingManager, wm *streaming.WebRTCManager, cm *chunking.ChunkManager) Model {
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
		cm:          cm,
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
		waitForWebRTCData(m.wm.DataChan),        // Listen for incoming chunks
		waitForChunkStatus(m.cm.StatusChan),     // Listen for hashing/parsing status
		waitForChunkProgress(m.cm.ProgressChan), // Listen for 0-100% progress
		waitForChunkError(m.cm.ErrChan),         // Listen for IO/Hashing errors
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

func waitForWebRTCData(sub <-chan []byte) tea.Cmd {
	return func() tea.Msg {
		data, ok := <-sub
		if !ok {
			return nil
		}
		return webrtcDataMsg{data: data}
	}
}

func waitForChunkStatus(sub <-chan string) tea.Cmd {
	return func() tea.Msg {
		status, ok := <-sub
		if !ok {
			return nil
		}
		return chunkStatusMsg{status: status}
	}
}

func waitForChunkProgress(sub <-chan int) tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-sub
		if !ok {
			return nil
		}
		return chunkProgressMsg{progress: progress}
	}
}

func waitForChunkError(sub <-chan error) tea.Cmd {
	return func() tea.Msg {
		err, ok := <-sub
		if !ok {
			return nil
		}
		return chunkErrorMsg{err: err}
	}
}
