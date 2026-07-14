package chunking

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func (cm *ChunkManager) SetOutDir(dir string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.OutDir = dir
}

func (cm *ChunkManager) ProcessIncomingMessage(msg []byte) {
	if len(msg) < 5 {
		return // Invalid packet
	}

	t := msg[0]
	length := binary.BigEndian.Uint32(msg[1:5])
	if uint32(len(msg)-5) < length {
		cm.ErrChan <- errors.New("incomplete TLV packet received")
		return
	}
	payload := msg[5 : 5+length]

	cm.mu.Lock()
	defer cm.mu.Unlock()

	switch t {
	case TypeMetadata:
		if cm.OutDir == "" {
			cm.ErrChan <- errors.New("incoming file rejected: output directory not set by user")
			return
		}

		cm.expectedHash = payload[0:32]
		cm.filename = string(payload[40:])

		if err := os.MkdirAll(cm.OutDir, 0o755); err != nil {
			cm.ErrChan <- fmt.Errorf("failed to create directories: %w", err)
			return
		}

		outPath := filepath.Join(cm.OutDir, cm.filename)
		f, err := os.Create(outPath)
		if err != nil {
			cm.ErrChan <- fmt.Errorf("file creation failed: %w", err)
			return
		}

		cm.currentFile = f
		cm.currentHash = sha256.New()
		cm.StatusChan <- fmt.Sprintf("Started receiving: %s", cm.filename)

	case TypeChunk:
		if cm.currentFile == nil {
			return
		}
		data := payload[4:] // Skip sequence number
		cm.currentFile.Write(data)
		cm.currentHash.Write(data)

	case TypeEOF:
		if cm.currentFile == nil {
			return
		}
		cm.currentFile.Close()

		finalHash := cm.currentHash.Sum(nil)
		if string(finalHash) != string(cm.expectedHash) {
			os.Remove(cm.currentFile.Name())
			cm.ErrChan <- fmt.Errorf("SHA256 mismatch: file %s corrupted", cm.filename)
		} else {
			cm.StatusChan <- fmt.Sprintf("File %s received and verified successfully!", cm.filename)
		}
		cm.currentFile = nil // Reset state
	}
}
