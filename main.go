package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/peterjohnbishop/symmetrical-stream/signaling"
	"github.com/peterjohnbishop/symmetrical-stream/streaming"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	mux := &sync.RWMutex{}

	ss := &signaling.SignalingManager{
		Mux:         mux,
		Identifier:  GenerateIdentifier(),
		MessageChan: make(chan signaling.EventMessage, 100),
		ErrChan:     make(chan error, 100),
	}
	ss.ConnectToSignalingServer()
	go ss.StartListening()

	wrtc := &streaming.WebRTCManager{
		Mux:         mux,
		WC:          ss.Conn,
		ErrChan:     ss.ErrChan,
		StatusChan:  make(chan string, 100),
		DataChan:    make(chan []byte, 1024),
		MessageChan: make(chan []byte, 1024),
	}

	RouteWebRTCToServer(wrtc, ss)

	select {}
}

func GenerateIdentifier() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", r.Intn(1000000))
}

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
