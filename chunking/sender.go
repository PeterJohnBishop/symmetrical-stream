// Package chunking handles file chunking
package chunking

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (cm *ChunkManager) SendFile(filePath string, transmitFunc func([]byte) error, waitBufferFunc func()) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	totalSize := stat.Size()

	cm.StatusChan <- fmt.Sprintf("Hashing file %s...", filepath.Base(filePath))
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return err
	}
	fullHash := h.Sum(nil)
	file.Seek(0, 0)

	filename := []byte(filepath.Base(filePath))
	meta := make([]byte, 40+len(filename))
	copy(meta[0:32], fullHash)
	binary.BigEndian.PutUint64(meta[32:40], uint64(totalSize))
	copy(meta[40:], filename)

	if err := transmitFunc(encodeTLV(TypeMetadata, meta)); err != nil {
		return err
	}

	cm.StatusChan <- "Transmitting file data..."
	buf := make([]byte, cm.ChunkSize)
	var seq uint32 = 0
	var bytesSent int64 = 0

	for {
		n, err := file.Read(buf)

		// Process bytes FIRST if any were read, even if EOF is returned simultaneously
		if n > 0 {
			waitBufferFunc()

			payload := make([]byte, 4+n)
			binary.BigEndian.PutUint32(payload[0:4], seq)
			copy(payload[4:], buf[:n])

			if errTransmit := transmitFunc(encodeTLV(TypeChunk, payload)); errTransmit != nil {
				return errTransmit
			}

			seq++
			bytesSent += int64(n)

			progress := int((float64(bytesSent) / float64(totalSize)) * 100)
			select {
			case cm.ProgressChan <- progress:
			default:
			}
		}

		// Check for EOF only AFTER processing the final chunk
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	cm.StatusChan <- "Transmission complete. Sending EOF..."
	return transmitFunc(encodeTLV(TypeEOF, nil))
}

func encodeTLV(t byte, payload []byte) []byte {
	length := uint32(len(payload))
	buf := make([]byte, 5+length)
	buf[0] = t
	binary.BigEndian.PutUint32(buf[1:5], length)
	if length > 0 {
		copy(buf[5:], payload)
	}
	return buf
}
