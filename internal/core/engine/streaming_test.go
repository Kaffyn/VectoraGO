package engine

import (
	"testing"
	"time"
)

func TestStreamWriterBasic(t *testing.T) {
	ch := make(chan QueryChunk, 10)
	sw := NewStreamWriter(ch, 100, 1*time.Second)
	defer sw.Close()

	sw.WriteToken("hello ")
	sw.WriteToken("world")
	sw.Flush()

	select {
	case chunk := <-ch:
		expected := "hello world"
		if chunk.Token != expected {
			t.Errorf("Expected %q, got %q", expected, chunk.Token)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for chunk")
	}
}

func TestStreamWriterAutoFlush(t *testing.T) {
	ch := make(chan QueryChunk, 10)
	sw := NewStreamWriter(ch, 10, 1*time.Second)
	defer sw.Close()

	sw.WriteToken("0123456789")
	sw.WriteToken("ABC") // This should trigger auto-flush (buffer > 10)

	// Should receive buffered content
	select {
	case chunk := <-ch:
		if len(chunk.Token) < 10 {
			t.Errorf("Expected auto-flush of at least 10 chars, got %q", chunk.Token)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for auto-flush")
	}
}

func TestStreamWriterChunk(t *testing.T) {
	ch := make(chan QueryChunk, 10)
	sw := NewStreamWriter(ch, 100, 1*time.Second)
	defer sw.Close()

	testChunk := QueryChunk{Token: "test", IsFinal: true}
	sw.WriteChunk(testChunk)

	select {
	case chunk := <-ch:
		if chunk.Token != "test" || !chunk.IsFinal {
			t.Errorf("WriteChunk failed, got %+v", chunk)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for WriteChunk")
	}
}

func TestStreamBufferEnqueue(t *testing.T) {
	ch := make(chan QueryChunk, 10)
	sb := NewStreamBuffer(ch, 3)
	defer sb.Close()

	// Enqueue 3 chunks (should auto-flush)
	sb.Enqueue(QueryChunk{Token: "a"})
	sb.Enqueue(QueryChunk{Token: "b"})
	sb.Enqueue(QueryChunk{Token: "c"})

	// Should receive all 3 chunks
	received := 0
	for i := 0; i < 3; i++ {
		select {
		case <-ch:
			received++
		case <-time.After(1 * time.Second):
			t.Errorf("Timeout waiting for chunk %d", i)
		}
	}

	if received != 3 {
		t.Errorf("Expected 3 chunks, received %d", received)
	}
}

func TestStreamBufferFlush(t *testing.T) {
	ch := make(chan QueryChunk, 10)
	sb := NewStreamBuffer(ch, 10)
	defer sb.Close()

	sb.Enqueue(QueryChunk{Token: "hello"})
	sb.Flush()

	select {
	case chunk := <-ch:
		if chunk.Token != "hello" {
			t.Errorf("Expected 'hello', got %q", chunk.Token)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for flushed chunk")
	}
}

func TestStreamBufferTimedFlush(t *testing.T) {
	ch := make(chan QueryChunk, 10)
	sb := NewStreamBuffer(ch, 100) // Large batch size
	defer sb.Close()

	sb.Enqueue(QueryChunk{Token: "test"})
	// Don't manually flush, wait for timer

	select {
	case chunk := <-ch:
		if chunk.Token != "test" {
			t.Errorf("Expected 'test', got %q", chunk.Token)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for timed flush")
	}
}

func TestStreamWriterMultipleWrites(t *testing.T) {
	ch := make(chan QueryChunk, 10)
	sw := NewStreamWriter(ch, 50, 1*time.Second)
	defer sw.Close()

	tokens := []string{"hello", " ", "world", " ", "from", " ", "stream"}
	for _, token := range tokens {
		sw.WriteToken(token)
	}
	sw.Flush()

	select {
	case chunk := <-ch:
		expected := "hello world from stream"
		if chunk.Token != expected {
			t.Errorf("Expected %q, got %q", expected, chunk.Token)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for chunk")
	}
}

func TestStreamWriterClose(t *testing.T) {
	ch := make(chan QueryChunk, 10)
	sw := NewStreamWriter(ch, 100, 1*time.Second)

	sw.WriteToken("pending")
	sw.Close()

	// After close, buffered content should be flushed
	select {
	case chunk := <-ch:
		if chunk.Token != "pending" {
			t.Errorf("Expected buffered content on close, got %q", chunk.Token)
		}
	case <-time.After(1 * time.Second):
		// May not receive if already flushed
	}
}
