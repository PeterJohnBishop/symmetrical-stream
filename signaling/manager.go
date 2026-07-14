package signaling

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type SignalingManager struct {
	Mux         *sync.RWMutex
	Conn        *websocket.Conn
	Identifier  string
	Receiver    string
	MessageChan chan EventMessage
	ErrChan     chan error
}

type EventMessage struct {
	Type    string          `json:"type"`
	Sender  string          `json:"sender"`
	Target  string          `json:"target,omitempty"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}
