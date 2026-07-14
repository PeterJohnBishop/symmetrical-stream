package streaming

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type WebRTCManager struct {
	Mux         *sync.RWMutex
	WC          *websocket.Conn
	PC          *webrtc.PeerConnection
	DC          *webrtc.DataChannel
	StatusChan  chan string
	ErrChan     chan error
	DataChan    chan []byte
	MessageChan chan []byte
	Identifier  string
	Receiver    string
}

type EventMessage struct {
	Type    string          `json:"type"`
	Sender  string          `json:"sender"`
	Target  string          `json:"target,omitempty"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}
