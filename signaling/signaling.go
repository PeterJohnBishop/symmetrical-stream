// Package signaling handles websocket comms
package signaling

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var DefaultServerURL = "localhost:8080"

// ConnectToSignalingServer establishes the websocket connection to the signaling server.
func (s *SignalingManager) ConnectToSignalingServer() {
	host := os.Getenv("HOST")
	if host == "" {
		host = DefaultServerURL
	}
	scheme := "wss"
	if strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		scheme = "ws"
	}
	u := url.URL{Scheme: scheme, Host: host, Path: "/ws"}
	var err error
	s.Conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		select {
		case s.ErrChan <- fmt.Errorf("[ERROR]failed to reach the signaling server: %w", err):
		default:
			fmt.Printf("[ERROR] Dropped error message to avoid blocking: %v\n", err)
		}
		return
	}
}

// StartListening listens for EventMessages
func (s *SignalingManager) StartListening() {
	defer close(s.MessageChan)

	s.Mux.Lock()
	if s.Conn != nil {
		s.Conn.SetReadDeadline(time.Time{})
	}
	s.Mux.Unlock()

	for {
		if s.Conn == nil {
			return
		}

		_, rawMsg, err := s.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				select {
				case s.ErrChan <- fmt.Errorf("websocket connection lost unexpectedly: %w", err):
				default:
				}
			}
			return
		}

		var msg EventMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			fmt.Printf("[WARN] Failed to parse signaling payload: %v\n", err)
			continue
		}

		select {
		case s.MessageChan <- msg:
		default:
		}
	}
}

// SendEventMessage sends an EventMessage
func (s *SignalingManager) SendEventMessage(eventType string, msgContent string, target *string, rawData ...json.RawMessage) {
	var targetVal string
	if target != nil {
		targetVal = *target
	}

	event := EventMessage{
		Type:    eventType,
		Message: msgContent,
		Sender:  s.Identifier,
		Target:  targetVal,
	}
	if len(rawData) > 0 {
		event.Data = rawData[0]
	}

	s.Mux.Lock()
	if s.Conn == nil {
		s.Mux.Unlock()
		return
	}
	err := s.Conn.WriteJSON(event)
	s.Mux.Unlock()

	if err != nil {
		select {
		case s.ErrChan <- fmt.Errorf("failed to send event %s: %w", eventType, err):
		default:
			fmt.Printf("[ERROR] Dropped error message to avoid blocking: %v\n", err)
		}
		return
	}
}

// RegisterWithSignalingServer sends an EventMessage specifically for registering the device with the signaling server, making it discoverable by other devices.
func (s *SignalingManager) RegisterWithSignalingServer() {
	event := EventMessage{
		Type:    "connect",
		Message: "Registering device with the signaling server",
		Sender:  s.Identifier,
	}

	s.Mux.Lock()
	if s.Conn == nil {
		s.Mux.Unlock()
		return
	}
	err := s.Conn.WriteJSON(event)
	s.Mux.Unlock()

	if err != nil {
		select {
		case s.ErrChan <- fmt.Errorf("[ERROR]failed to register with the signaling server: %w", err):
		default:
			fmt.Printf("[ERROR] Dropped error message to avoid blocking: %v\n", err)
		}
		return
	}
}
