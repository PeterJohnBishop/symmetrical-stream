package chunking

import (
	"hash"
	"os"
	"sync"
)

const (
	TypeMetadata     byte = 1
	TypeChunk        byte = 2
	TypeEOF          byte = 3
	DefaultChunkSize      = 16 * 1024 // 16KB
)

type ChunkManager struct {
	ChunkSize int
	OutDir    string

	mu           sync.Mutex
	currentFile  *os.File
	currentHash  hash.Hash
	expectedHash []byte
	filename     string
	expectedSeq  uint32

	ProgressChan chan int
	StatusChan   chan string
	ErrChan      chan error
}
