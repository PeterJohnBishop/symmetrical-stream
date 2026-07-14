// Package tui handles the terminal presentation of the app
package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/peterjohnbishop/symmetrical-stream/signaling"
)

type (
	connectedMsg struct{}
	eventMsg     struct{ msg signaling.EventMessage }
	errorMsg     struct{ err error }
)

type Model struct {
	sm     *signaling.SignalingManager
	status string
	logs   []string
}

func InitialModel(sm *signaling.SignalingManager) Model {
	return Model{
		sm:     sm,
		status: "Initializing...",
		logs:   []string{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.connectCmd(),
		waitForMessage(m.sm.MessageChan),
		waitForError(m.sm.ErrChan),
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
