// Package signaling handles websocket comms
package signaling

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var mux sync.RWMutex

type EventMessage struct {
	Type    string          `json:"type"`
	Sender  string          `json:"sender"`
	Target  string          `json:"target,omitempty"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ConnectToSignalingServer establishes the websocket connection to the signaling server.
func ConnectToSignalingServer() (*websocket.Conn, error) {
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost:8080"
	}
	scheme := "wss"
	if strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		scheme = "ws"
	}
	u := url.URL{Scheme: scheme, Host: host, Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// SendEventMessage sends an event, message, and the senderID to the signaling server. Target and rawData are optional. Errors are sent to the errChan.
func SendEventMessage(conn *websocket.Conn, errChan chan error, eventType string, msgContent string, sender string, target *string, rawData ...json.RawMessage) {
	var targetVal string
	if target != nil {
		targetVal = *target
	}

	event := EventMessage{
		Type:    eventType,
		Message: msgContent,
		Sender:  sender,
		Target:  targetVal,
	}
	if len(rawData) > 0 {
		event.Data = rawData[0]
	}

	mux.Lock()
	if conn == nil {
		mux.Unlock()
		return
	}
	err := conn.WriteJSON(event)
	mux.Unlock()

	if err != nil {
		select {
		case errChan <- fmt.Errorf("failed to send event %s: %w", eventType, err):
		default:
			fmt.Printf("[ERROR] Dropped error message to avoid blocking: %v\n", err)
		}
	}
}

// RegisterWithSignalingServer sends an EventMessage specifically for registering the device with the signaling server, making it discoverable by other devices.
func RegisterWithSignalingServer(conn *websocket.Conn, errChan chan error, sender string) {

	event := EventMessage{
		Type:    "connect",
		Message: "Registering device with the signaling server",
		Sender:  sender,
	}

	mux.Lock()
	if conn == nil {
		mux.Unlock()
		return
	}
	err := conn.WriteJSON(event)
	mux.Unlock()

	if err != nil {
		select {
		case errChan <- fmt.Errorf("[ERROR]failed to register with the signaling server: %w", err):
		default:
			fmt.Printf("[ERROR] Dropped error message to avoid blocking: %v\n", err)
		}
	}
}
