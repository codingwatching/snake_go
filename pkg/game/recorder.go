package game

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// GameRecorder handles asynchronous logging of game steps
type GameRecorder struct {
	file       *os.File
	writer     *bufio.Writer
	recordChan chan StepRecord
	wg         sync.WaitGroup
	mu         sync.Mutex
	closed     bool
}

// NewRecorder creates a new recorder that writes to records/ directory
// Filename format: game_{sessionID}_{timestamp}.jsonl
func NewRecorder(sessionID string) (*GameRecorder, error) {
	// Ensure records directory exists
	recordDir := "data/records"
	if err := os.MkdirAll(recordDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create records dir: %w", err)
	}

	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("game_%s_%d.jsonl", sessionID, timestamp)
	path := filepath.Join(recordDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create record file: %w", err)
	}

	r := &GameRecorder{
		file:       f,
		writer:     bufio.NewWriter(f),
		recordChan: make(chan StepRecord, 1000), // Buffer up to 1000 frames
	}

	// Start background writer
	r.wg.Add(1)
	go r.writeLoop()

	return r, nil
}

// RecordStep queues a record to be written. Non-blocking (drops if full).
func (r *GameRecorder) RecordStep(rec StepRecord) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()

	select {
	case r.recordChan <- rec:
		// Queued successfully
	default:
		// Channel full, drop frame to protect game loop performance
		// In a real production environment, maybe log a warning metric
	}
}

// Close flushes the buffer and closes the file
func (r *GameRecorder) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	r.mu.Unlock()

	close(r.recordChan)
	r.wg.Wait() // Wait for writeLoop to finish
	r.file.Close()
}

func (r *GameRecorder) writeLoop() {
	defer r.wg.Done()

	encoder := json.NewEncoder(r.writer)
	for rec := range r.recordChan {
		if err := encoder.Encode(rec); err != nil {
			fmt.Fprintf(os.Stderr, "Error recording frame: %v\n", err)
			continue
		}
	}
	r.writer.Flush()
}
