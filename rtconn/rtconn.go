package rtconn

import (
	"encoding/json"
	"fmt"

	"github.com/peterjohnbishop/symmetrical-stream/wsconn"
	"github.com/pion/webrtc/v4"
)

type WebRTCManager struct {
	PC             *webrtc.PeerConnection
	DC             *webrtc.DataChannel
	WC             *wsconn.ConnectionManager
	StatusChan     chan string
	LocalDataChan  chan string
	RemoteDataChan chan string
}

func (m *WebRTCManager) StartWebRTC() {
	if m.WC == nil {
		m.sendStatus("Connection manager is not initialized. Cannot start WebRTC.")
		return
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		m.sendStatus(fmt.Sprintf("failed to create peer connection: %w", err))
	}
	m.PC = pc

	dc, err := m.PC.CreateDataChannel("dataTransfer", nil)
	if err != nil {
		m.sendStatus(fmt.Sprintf("failed to create data channel: %w", err))
		return
	}
	m.DC = dc

	dc.OnOpen(func() {
		m.sendStatus("Data channel is open! Starting ASCII stream...")
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		// Handle incoming messages from the peer
	})

	m.PC.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			candidateJSON := candidate.ToJSON()

			candidateBytes, err := json.Marshal(candidateJSON)
			if err != nil {
				m.sendStatus(fmt.Sprintf("Failed to marshal ICE candidate: %v", err))
				return
			}

			m.WC.SendEventMessage("candidate", "ICE Candidate", nil, candidateBytes)
		}
	})

	m.PC.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		m.sendStatus(fmt.Sprintf("ICE Connection State has changed: %s", connectionState.String()))
		if connectionState == webrtc.ICEConnectionStateConnected {
			// Connection established, you can start sending data or streaming
		}
	})

	m.PC.OnDataChannel(func(d *webrtc.DataChannel) {
		m.sendStatus("Incoming data channel received from peer!")
		m.DC = d

		d.OnOpen(func() {
			// Data channel is open, you can start sending data or streaming
		})

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			// Handle incoming messages from the peer
		})
	})

	m.sendStatus("WebRTC is ready to connect. Searching for ICE candidates...")
}

func (m *WebRTCManager) sendStatus(msg string) {
	if m.StatusChan != nil {
		select {
		case m.StatusChan <- msg:
		default:
		}
	}
}

// SendOffer creates a WebRTC offer and sends it to the specified target via the signaling server.
func (m *WebRTCManager) SendOffer(target string) {
	if m.PC == nil {
		m.sendStatus("Peer connection is nil. Call StartWebRTC first")
		return
	}

	offer, err := m.PC.CreateOffer(nil)
	if err != nil {
		m.sendStatus(fmt.Sprintf("failed to create offer: %w", err))
		return
	}

	if err := m.PC.SetLocalDescription(offer); err != nil {
		m.sendStatus(fmt.Sprintf("failed to set local description: %w", err))
		return
	}

	offerBytes, err := json.Marshal(offer)
	if err != nil {
		m.sendStatus(fmt.Sprintf("failed to marshal offer: %w", err))
		return
	}

	m.WC.SendEventMessage("offer", "WebRTC Offer", &target, offerBytes)

	m.sendStatus("Outbound offer generated and sent to signaling server")
}

// HandleOffer processes an incoming WebRTC offer, sets the remote description,
func (m *WebRTCManager) HandleOffer(sender string, remoteSDP string) {
	if m.PC == nil {
		m.sendStatus("Peer connection not initialized")
	}

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: remoteSDP}
	if err := m.PC.SetRemoteDescription(offer); err != nil {
		m.sendStatus(fmt.Sprintf("failed to set remote description: %w", err))
		return
	}

	answer, err := m.PC.CreateAnswer(nil)
	if err != nil {
		m.sendStatus(fmt.Sprintf("failed to create answer: %w", err))
		return
	}

	if err := m.PC.SetLocalDescription(answer); err != nil {
		m.sendStatus(fmt.Sprintf("failed to set local description: %w", err))
		return
	}

	answerBytes, _ := json.Marshal(answer)
	m.WC.SendEventMessage("answer", "WebRTC Answer", &sender, answerBytes)

	m.sendStatus("Offer accepted. Outbound answer sent.")
}

// HandleAnswer processes an incoming WebRTC answer and sets it as the remote description to complete the handshake.
func (m *WebRTCManager) HandleAnswer(remoteSDP string) {
	if m.PC == nil {
		m.sendStatus("Peer connection not initialized")
		return
	}

	answer := webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: remoteSDP}

	if err := m.PC.SetRemoteDescription(answer); err != nil {
		m.sendStatus(fmt.Sprintf("failed to apply remote answer: %w", err))
		return
	}

	m.sendStatus("Handshake complete for P2P tunnel.")
}

// SentTextMessage sends a text message over the established WebRTC data channel.
func (m *WebRTCManager) SendTextMessage(text string) {
	if m.DC == nil || m.DC.ReadyState() != webrtc.DataChannelStateOpen {
		m.sendStatus("Data channel is not open")
		return
	}
	m.sendStatus(fmt.Sprintf("[TEXT] -> %s", text))
	m.DC.SendText(text)
}

// SendBinaryData sends binary data over the established WebRTC data channel.
func (m *WebRTCManager) SendBinaryData(data []byte) {
	if m.DC == nil || m.DC.ReadyState() != webrtc.DataChannelStateOpen {
		m.sendStatus("Data channel is not open")
		return
	}

	m.sendStatus(fmt.Sprintf("[BINARY] -> Sending %d bytes", len(data)))
	m.DC.Send(data)
}

// Disconnect safely closes the WebRTC connection and Data Channel
func (m *WebRTCManager) Disconnect() {
	if m.DC != nil {
		m.DC.Close()
	}
	if m.PC != nil {
		m.PC.Close()
	}
	m.PC = nil
	m.DC = nil
	m.sendStatus("WebRTC connection closed")
}
