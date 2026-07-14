package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/joho/godotenv"
	"github.com/peterjohnbishop/symmetrical-stream/chunking"
	"github.com/peterjohnbishop/symmetrical-stream/signaling"
	"github.com/peterjohnbishop/symmetrical-stream/streaming"
	"github.com/peterjohnbishop/symmetrical-stream/tui"
)

func main() {
	// load the websocket/signaling server address
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	mux := &sync.RWMutex{}

	// initialize the SignalingManager, make the inital connection, and start listening for events
	ss := &signaling.SignalingManager{
		Mux:         mux,
		Identifier:  GenerateIdentifier(),
		MessageChan: make(chan signaling.EventMessage, 100),
		ErrChan:     make(chan error, 100),
	}
	ss.ConnectToSignalingServer()
	go ss.StartListening()

	// initializer the WebRTCManager, and start up the background process to pipe messages from the WebRTCManager to the SignalingManager
	wrtc := &streaming.WebRTCManager{
		Mux:         mux,
		WC:          ss.Conn,
		ErrChan:     ss.ErrChan,
		StatusChan:  make(chan string, 100),
		DataChan:    make(chan []byte, 1024),
		MessageChan: make(chan []byte, 1024),
	}

	RouteWebRTCToServer(wrtc, ss)

	ck := &chunking.ChunkManager{
		ChunkSize:    chunking.DefaultChunkSize,
		ProgressChan: make(chan int, 100),
		StatusChan:   make(chan string, 100),
		ErrChan:      make(chan error, 100),
	}

	// data sent through the webrtc data channel are sent to ProcessIncomingMessage
	go func() {
		for msg := range wrtc.DataChan {
			ck.ProcessIncomingMessage(msg)
		}
	}()

	app := tui.InitialModel(ss)
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error starting the TUI: %v", err)
		os.Exit(1)
	}
}

// GenerateIdentifier generates a time based 6 digit value as a unique identifier
func GenerateIdentifier() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", r.Intn(1000000))
}

// RouteWebRTCToServer takes streaming.EventMessage data as []byte, unmarshals the data as signaling.EventMessage, and pipes it into the SignalingManager
func RouteWebRTCToServer(w *streaming.WebRTCManager, s *signaling.SignalingManager) {
	go func() {
		for rawBytes := range w.MessageChan {
			var msg signaling.EventMessage
			if err := json.Unmarshal(rawBytes, &msg); err != nil {
				select {
				case s.ErrChan <- fmt.Errorf("failed to unmarshal outbound WebRTC message: %w", err):
				default:
					fmt.Printf("[WARN] Dropped unmarshal error to avoid blocking: %v\n", err)
				}
				continue
			}
			s.SendEventMessage(msg.Type, msg.Message, &msg.Target, msg.Data)
		}
	}()
}
