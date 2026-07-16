// Package streaming handles the WebRTC connection
package streaming

import (
	"encoding/json"
	"fmt"

	"github.com/pion/webrtc/v4"
)

// StartWebRTC initializes the webRTC PeerConnection, Data Channel, and callbacks
func (w *WebRTCManager) StartWebRTC(isSender bool) {
	var err error
	if w.WC == nil {
		select {
		case w.ErrChan <- fmt.Errorf("connection manager must have an initialized websocket"):
		default:
			fmt.Printf("[ERROR] Dropped error message to avoid blocking\n")
		}
		return
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	}

	w.PC, err = webrtc.NewPeerConnection(config)
	if err != nil {
		select {
		case w.ErrChan <- fmt.Errorf("failed to create peer connection: %w", err):
		default:
			fmt.Printf("[ERROR] Dropped error message to avoid blocking\n")
		}
		return
	}

	if isSender {
		w.DC, err = w.PC.CreateDataChannel("dataTransfer", nil)
		if err != nil {
			select {
			case w.ErrChan <- fmt.Errorf("failed to create data channel: %w", err):
			default:
				fmt.Printf("[ERROR] Dropped error message to avoid blocking\n")
			}
			return
		}

		w.DC.OnOpen(func() {
			if w.StatusChan != nil {
				select {
				case w.StatusChan <- "Local Data channel is open. Sending...":
				default:
				}
			}
		})

		w.DC.OnMessage(func(msg webrtc.DataChannelMessage) {
			if w.DataChan != nil {
				safeData := make([]byte, len(msg.Data))
				copy(safeData, msg.Data)

				select {
				case w.DataChan <- safeData:
				default:
				}
			}
		})
	} else {
		w.PC.OnDataChannel(func(d *webrtc.DataChannel) {
			w.Mux.Lock()
			w.DC = d
			w.Mux.Unlock()

			d.OnOpen(func() {
				if w.StatusChan != nil {
					select {
					case w.StatusChan <- "Remote Data channel opened. Ready to receive...":
					default:
					}
				}
			})

			d.OnMessage(func(msg webrtc.DataChannelMessage) {
				if w.DataChan != nil {
					safeData := make([]byte, len(msg.Data))
					copy(safeData, msg.Data)

					select {
					case w.DataChan <- safeData:
					default:
					}
				}
			})
		})
	}

	w.PC.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		candidateJSON := candidate.ToJSON()
		candidateBytes, err := json.Marshal(candidateJSON)
		if err != nil {
			if w.StatusChan != nil {
				select {
				case w.StatusChan <- fmt.Sprintf("Failed to marshal ICE candidate: %v", err):
				default:
				}
			}
			return
		}

		msg := EventMessage{
			Type:    "candidate",
			Sender:  w.Identifier,
			Message: "ICE Candidate",
			Target:  w.Receiver,
			Data:    candidateBytes,
		}

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			if w.StatusChan != nil {
				select {
				case w.StatusChan <- fmt.Sprintf("Failed to marshal EventMessage: %v", err):
				default:
				}
			}
		}

		w.MessageChan <- msgBytes
	})

	w.PC.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		if w.StatusChan != nil {
			select {
			case w.StatusChan <- fmt.Sprintf("ICE Connection State has changed: %s", connectionState.String()):
			default:
			}
		}
		if connectionState == webrtc.ICEConnectionStateConnected {
			if w.StatusChan != nil {
				select {
				case w.StatusChan <- "Peers connected!":
				default:
				}
			}
		}
	})

	if w.StatusChan != nil {
		select {
		case w.StatusChan <- "WebRTC is ready to connect. Searching for ICE candidates...":
		default:
		}
	}
}

// HandleICECandidate unmarshals inncomming ICE candidates to find the best connection option
func (w *WebRTCManager) HandleICECandidate(candidateBytes []byte) error {
	w.Mux.RLock()
	pc := w.PC
	w.Mux.RUnlock()

	if pc == nil {
		return fmt.Errorf("peer connection must be initialized before adding candidates")
	}

	var candidate webrtc.ICECandidateInit
	if err := json.Unmarshal(candidateBytes, &candidate); err != nil {
		return fmt.Errorf("failed to unmarshal remote ICE candidate: %w", err)
	}

	if err := pc.AddICECandidate(candidate); err != nil {
		return fmt.Errorf("failed to add remote ICE candidate: %w", err)
	}

	if w.StatusChan != nil {
		select {
		case w.StatusChan <- "Remote ICE candidate applied successfully.":
		default:
		}
	}
	return nil
}

// SendOffer creates and sends the Offer to start the webrtc handshake
func (w *WebRTCManager) SendOffer(target string) error {
	w.Mux.RLock()
	pc := w.PC
	w.Mux.RUnlock()

	if pc == nil {
		return fmt.Errorf("peer connection is nil. call StartWebRTC first")
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create an offer: %w", err)
	}

	if err := pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	offerBytes, err := json.Marshal(offer)
	if err != nil {
		return fmt.Errorf("failed to marshal offer: %w", err)
	}

	msg := EventMessage{
		Type:    "offer",
		Sender:  w.Identifier,
		Message: "WebRTC",
		Target:  target,
		Data:    offerBytes,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal event message: %w", err)
	}

	w.MessageChan <- msgBytes

	if w.StatusChan != nil {
		select {
		case w.StatusChan <- "outbound offer generated and sent to signaling server.":
		default:
		}
	}
	return nil
}

// HandleOffer applies the remote description and sends an Answer in response to the Offer
func (w *WebRTCManager) HandleOffer(sender string, offerBytes []byte) error {
	w.Mux.RLock()
	pc := w.PC
	w.Mux.RUnlock()

	if pc == nil {
		return fmt.Errorf("peer connection must be initialized")
	}

	var offer webrtc.SessionDescription
	if err := json.Unmarshal(offerBytes, &offer); err != nil {
		return fmt.Errorf("failed to unmarshal remote offer: %w", err)
	}

	if err := pc.SetRemoteDescription(offer); err != nil {
		return fmt.Errorf("failed to set the session description: %w", err)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("failed to create an answer: %w", err)
	}

	if err := pc.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("failed to set the local description: %w", err)
	}

	answerBytes, err := json.Marshal(answer)
	if err != nil {
		return fmt.Errorf("failed to marshal answer: %w", err)
	}

	msg := EventMessage{
		Type:    "answer",
		Sender:  w.Identifier,
		Message: "WebRTC Answer",
		Target:  sender,
		Data:    answerBytes,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal event message: %w", err)
	}

	w.MessageChan <- msgBytes

	if w.StatusChan != nil {
		select {
		case w.StatusChan <- "offer accepted. outbound answer sent.":
		default:
		}
	}
	return nil
}

// HandleAnswer applies the remote description to complete the webrtc handshake
func (w *WebRTCManager) HandleAnswer(answerBytes []byte) error {
	w.Mux.RLock()
	pc := w.PC
	w.Mux.RUnlock()

	if pc == nil {
		return fmt.Errorf("peer connection must be initialized")
	}

	var answer webrtc.SessionDescription
	if err := json.Unmarshal(answerBytes, &answer); err != nil {
		return fmt.Errorf("failed to unmarshal remote answer: %w", err)
	}

	if err := pc.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("failed to apply remote answer: %w", err)
	}

	if w.StatusChan != nil {
		select {
		case w.StatusChan <- "handshake complete for P2P tunnel":
		default:
		}
	}
	return nil
}

// SafeWriteBytesToDC sends []byte data through the data channel
func (w *WebRTCManager) SafeWriteBytesToDC(data []byte) error {
	w.Mux.RLock()
	defer w.Mux.RUnlock()

	if w.DC == nil {
		return fmt.Errorf("data channel is not initialized")
	}

	return w.DC.Send(data)
}

// DisconnectWebRTC closes the Data Channel and Peer connection, effectively closing the webrtc connection
func (w *WebRTCManager) DisconnectWebRTC() {
	w.Mux.Lock()
	defer w.Mux.Unlock()

	if w.DC != nil {
		w.DC.Close()
	}
	if w.PC != nil {
		w.PC.Close()
	}
	w.DC = nil
	w.PC = nil

	if w.StatusChan != nil {
		select {
		case w.StatusChan <- "WebRTC connection closed":
		default:
		}
	}
}
