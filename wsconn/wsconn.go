package wsconn

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

type ConnectionManager struct {
	Conn        *websocket.Conn
	Err         error
	MessageChan chan EventMessage
	StatusChan  chan string
	ID          string
}

type EventMessage struct {
	Type    string          `json:"type"`
	Sender  string          `json:"sender"`
	Target  string          `json:"target,omitempty"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Connect establishes a WebSocket connection to the signaling server using the host specified in the environment variable or defaults to localhost:8080. It returns the established connection or an error if the connection fails.
func (c *ConnectionManager) Connect() *websocket.Conn {
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost:8080"
		status := fmt.Sprintf("HOST environment variable not set, defaulting to %s\n", host)
		c.StatusChan <- status
	}
	scheme := "wss"
	if strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") {
		scheme = "ws"
	}
	u := url.URL{Scheme: scheme, Host: host, Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		status := fmt.Sprintf("Failed to connect to WebSocket server at %s: %v", u.String(), err)
		c.StatusChan <- status
		return nil
	}
	return conn
}

func (c *ConnectionManager) sendStatus(msg string) {
	if c.StatusChan != nil {
		select {
		case c.StatusChan <- msg:
		default:
		}
	}
}

// StartListening continuously reads messages from the WebSocket connection and sends them to the MessageChan. If an error occurs while reading, it sends the error to the ErrorChan and exits.
func (c *ConnectionManager) StartListening() {
	defer close(c.MessageChan)

	for {
		_, rawMsg, err := c.Conn.ReadMessage()
		if err != nil {
			status := fmt.Sprintf("connection closed or read error: %w", err)
			c.StatusChan <- status
			return
		}

		var msg EventMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			status := fmt.Sprintf("failed to unmarshal message: %w", err)
			c.StatusChan <- status
			continue
		}
		c.MessageChan <- msg
	}
}

// SendEventMessage sends an event message with the specified type, content, target, and optional raw data over the WebSocket connection. If an error occurs while sending, it sends the error to the ErrorChan.
func (c *ConnectionManager) SendEventMessage(eventType string, msgContent string, target *string, rawData ...json.RawMessage) {
	var targetVal string
	if target != nil {
		targetVal = *target
	}

	event := EventMessage{
		Type:    eventType,
		Message: msgContent,
		Sender:  c.ID,
		Target:  targetVal,
	}
	if len(rawData) > 0 {
		event.Data = rawData[0]
	}

	err := c.Conn.WriteJSON(event)
	if err != nil {
		status := fmt.Sprintf("Failed to send message: %v", err)
		c.StatusChan <- status
	}
}
