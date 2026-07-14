package streaming

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

func StartWebRTC(isSender bool, mux *sync.RWMutex, conn *websocket.Conn, statusChan chan string, dataChan chan []byte) (*webrtc.PeerConnection, *webrtc.DataChannel, error) {
	pc := &webrtc.PeerConnection{}
	dc := &webrtc.DataChannel{}
	var err error

	mux.RLock()
	wc := conn
	mux.RUnlock()

	if wc == nil {
		return nil, nil, fmt.Errorf("connection manager must have an initialized websocket")
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	}

	pc, err = webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	if isSender {
		dc, err := pc.CreateDataChannel("dataTransfer", nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create data channel: %w", err)
		}

		dc.OnOpen(func() {
			if statusChan != nil {
				select {
				case statusChan <- "Local Data channel is open. Sending...":
				default:
				}
			}
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			if dataChan != nil {
				select {
				case dataChan <- msg.Data:
				default:
				}
			}
		})
	} else {
		pc.OnDataChannel(func(d *webrtc.DataChannel) {
			mux.Lock()
			dc = d
			mux.Unlock()

			d.OnOpen(func() {
				if statusChan != nil {
					select {
					case statusChan <- "Remote Data channel opened. Ready to receive...":
					default:
					}
				}
			})

			d.OnMessage(func(msg webrtc.DataChannelMessage) {
				if dataChan != nil {
					select {
					case dataChan <- msg.Data:
					default:
					}
				}
			})
		})
	}

	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		candidateJSON := candidate.ToJSON()
		candidateBytes, err := json.Marshal(candidateJSON)
		if err != nil {
			if statusChan != nil {
				select {
				case statusChan <- fmt.Sprintf("Failed to marshal ICE candidate: %v", err):
				default:
				}
			}
			return
		}

		p.mu.RLock()
		target := p.ActivePeer
		p.mu.RUnlock()

		p.SendEventMessage("candidate", "ICE Candidate", &target, candidateBytes)
	})

	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		p.sendStatus(fmt.Sprintf("ICE Connection State has changed: %s", connectionState.String()))

		if connectionState == webrtc.ICEConnectionStateConnected {
			p.sendStatus("Peers connected!")
		}
	})

	p.sendStatus("WebRTC is ready to connect. Searching for ICE candidates...")
	return nil
}
