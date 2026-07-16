package chunking

import (
	"bytes"
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
		return
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
			cm.ErrChan <- errors.New("incoming file rejected: output directory not set")
			return
		}

		cm.expectedHash = make([]byte, 32)
		copy(cm.expectedHash, payload[0:32])
		cm.filename = string(payload[40:])

		// Reset the sequence tracker for a new file
		cm.expectedSeq = 0

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

		seq := binary.BigEndian.Uint32(payload[0:4])

		if seq != cm.expectedSeq {
			cm.ErrChan <- fmt.Errorf("CRITICAL PACKET LOSS: Expected chunk %d, but got chunk %d", cm.expectedSeq, seq)
		}

		cm.expectedSeq = seq + 1

		data := payload[4:]
		cm.currentFile.Write(data)
		cm.currentHash.Write(data)

	case TypeEOF:
		if cm.currentFile == nil {
			return
		}
		cm.currentFile.Close()

		finalHash := cm.currentHash.Sum(nil)

		if !bytes.Equal(finalHash, cm.expectedHash) {
			os.Remove(cm.currentFile.Name())
			cm.ErrChan <- fmt.Errorf("SHA256 mismatch: file %s corrupted", cm.filename)
		} else {
			cm.StatusChan <- fmt.Sprintf("File %s received and verified successfully!", cm.filename)
		}
		cm.currentFile = nil
	}
}
