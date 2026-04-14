package engine

import (
	"context"
	"sync"
	"time"
)

// StreamWriter provides buffered streaming with flushing capabilities
// Useful for batching small chunks and improving throughput
type StreamWriter struct {
	ch       chan QueryChunk
	mu       sync.Mutex
	buffer   []rune
	maxBuf   int
	flushTTL time.Duration
	ticker   *time.Ticker
	done     chan struct{}
}

// NewStreamWriter creates a buffered stream writer
func NewStreamWriter(ch chan QueryChunk, maxBuffer int, flushTTL time.Duration) *StreamWriter {
	sw := &StreamWriter{
		ch:       ch,
		buffer:   make([]rune, 0, maxBuffer),
		maxBuf:   maxBuffer,
		flushTTL: flushTTL,
		ticker:   time.NewTicker(flushTTL),
		done:     make(chan struct{}),
	}

	// Background flush goroutine
	go func() {
		for {
			select {
			case <-sw.ticker.C:
				sw.Flush()
			case <-sw.done:
				return
			}
		}
	}()

	return sw
}

// WriteToken writes a token to the stream buffer
// Automatically flushes if buffer exceeds maxBuf
func (sw *StreamWriter) WriteToken(token string) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.buffer = append(sw.buffer, []rune(token)...)

	if len(sw.buffer) >= sw.maxBuf {
		sw.flushLocked()
	}
}

// WriteChunk writes a complete chunk to the channel
func (sw *StreamWriter) WriteChunk(chunk QueryChunk) {
	sw.mu.Lock()
	sw.flushLocked()
	sw.mu.Unlock()

	select {
	case sw.ch <- chunk:
	case <-sw.done:
	}
}

// Flush sends any buffered content to the channel
func (sw *StreamWriter) Flush() {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.flushLocked()
}

// flushLocked sends buffered content without acquiring lock (caller must lock)
func (sw *StreamWriter) flushLocked() {
	if len(sw.buffer) == 0 {
		return
	}

	select {
	case sw.ch <- QueryChunk{Token: string(sw.buffer), IsFinal: false}:
		sw.buffer = sw.buffer[:0]
	case <-sw.done:
	}
}

// Close finalizes the stream writer and flushes any remaining content
func (sw *StreamWriter) Close() {
	sw.mu.Lock()
	sw.flushLocked()
	sw.mu.Unlock()

	sw.ticker.Stop()
	close(sw.done)
}

// StreamBuffer is a more sophisticated buffering strategy using time-based batching
type StreamBuffer struct {
	ch        chan QueryChunk
	chunks    []QueryChunk
	mu        sync.Mutex
	batchSize int
	flushCh   chan struct{}
	stopCh    chan struct{}
}

// NewStreamBuffer creates a batching stream buffer
func NewStreamBuffer(ch chan QueryChunk, batchSize int) *StreamBuffer {
	sb := &StreamBuffer{
		ch:        ch,
		chunks:    make([]QueryChunk, 0, batchSize),
		batchSize: batchSize,
		flushCh:   make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
	}

	// Background flusher
	go sb.flushLoop()

	return sb
}

// Enqueue adds a chunk to the buffer
// Automatically flushes if batch size reached
func (sb *StreamBuffer) Enqueue(chunk QueryChunk) error {
	sb.mu.Lock()
	sb.chunks = append(sb.chunks, chunk)
	shouldFlush := len(sb.chunks) >= sb.batchSize
	sb.mu.Unlock()

	if shouldFlush {
		select {
		case sb.flushCh <- struct{}{}:
		case <-sb.stopCh:
			return context.Canceled
		default:
		}
	}

	return nil
}

// Flush manually triggers a flush of buffered chunks
func (sb *StreamBuffer) Flush() error {
	select {
	case sb.flushCh <- struct{}{}:
		return nil
	case <-sb.stopCh:
		return context.Canceled
	default:
		return nil
	}
}

// flushLoop sends batches of chunks
func (sb *StreamBuffer) flushLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sb.flushCh:
			sb.sendBatch()

		case <-ticker.C:
			sb.sendBatch()

		case <-sb.stopCh:
			sb.sendBatch() // Final flush
			return
		}
	}
}

// sendBatch sends all buffered chunks
func (sb *StreamBuffer) sendBatch() {
	sb.mu.Lock()
	if len(sb.chunks) == 0 {
		sb.mu.Unlock()
		return
	}

	chunks := sb.chunks
	sb.chunks = make([]QueryChunk, 0, sb.batchSize)
	sb.mu.Unlock()

	for _, chunk := range chunks {
		select {
		case sb.ch <- chunk:
		case <-sb.stopCh:
			return
		}
	}
}

// Close flushes remaining chunks and stops the buffer
func (sb *StreamBuffer) Close() error {
	close(sb.stopCh)
	// Give time for final flush
	time.Sleep(10 * time.Millisecond)
	return nil
}
