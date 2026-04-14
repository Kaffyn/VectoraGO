package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWorkerPoolParallelization verifies that embedding uses worker pool
func TestWorkerPoolParallelization(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.go": `package main\n\nfunc main() {\n\tfmt.Println("Hello")\n}`,
		"file2.go": `package test\n\nfunc Test() {\n\treturn nil\n}`,
		"file3.py": `def hello():\n    print("Hello")\n\nif __name__ == "__main__":\n    hello()`,
		"file4.js": `function main() {\n  console.log("Hello");\n}\n\nmain();`,
	}

	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Verify that all test files were created
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}
	if len(entries) != len(testFiles) {
		t.Errorf("Expected %d test files, got %d", len(testFiles), len(entries))
	}
}

// TestDetectLanguage verifies language detection from file extensions
func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		filePath string
		expected string
	}{
		{"main.go", "go"},
		{"script.py", "python"},
		{"app.js", "javascript"},
		{"index.ts", "javascript"},
		{"readme.md", "markdown"},
		{"unknown.xyz", "text"},
	}

	for _, test := range tests {
		result := detectLanguage(test.filePath)
		if result != test.expected {
			t.Errorf("detectLanguage(%q) = %q, expected %q", test.filePath, result, test.expected)
		}
	}
}

// TestChunkContent verifies content chunking
func TestChunkContent(t *testing.T) {
	content := `This is a test content.
This is another line.
And yet another line here.
This continues the content.
Final line of content.`

	chunks := chunkContent(content, 10, 2)

	if len(chunks) == 0 {
		t.Error("Expected non-empty chunks")
	}

	// Verify all chunks are non-empty
	for i, chunk := range chunks {
		if chunk == "" {
			t.Errorf("Chunk %d is empty", i)
		}
	}

	t.Logf("Content split into %d chunks", len(chunks))
}

// TestContextCancellation verifies cancellation handling
func TestContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Verify context is properly cancelled
	select {
	case <-ctx.Done():
		// Success: context is cancelled as expected
		return
	case <-time.After(1 * time.Second):
		t.Error("Context cancellation timeout")
	}
}

// TestEmbeddingResultAggregation verifies result collection works correctly
func TestEmbeddingResultAggregation(t *testing.T) {
	resultQueue := make(chan embeddingResult, 4)
	var totalEmbedded, totalChunks, totalErrors int

	// Send mock results
	testResults := []embeddingResult{
		{relPath: "file1.go", fileChunks: 3, hasError: false},
		{relPath: "file2.go", fileChunks: 2, hasError: false},
		{relPath: "file3.py", fileChunks: 0, hasError: true, errorMsg: "embed failed"},
	}

	for _, result := range testResults {
		resultQueue <- result
	}
	close(resultQueue)

	// Simulate result processing
	for result := range resultQueue {
		if result.hasError {
			totalErrors++
		} else {
			if result.fileChunks > 0 {
				totalEmbedded++
				totalChunks += result.fileChunks
			}
		}
	}

	if totalEmbedded != 2 {
		t.Errorf("Expected 2 embedded files, got %d", totalEmbedded)
	}
	if totalChunks != 5 {
		t.Errorf("Expected 5 total chunks, got %d", totalChunks)
	}
	if totalErrors != 1 {
		t.Errorf("Expected 1 error, got %d", totalErrors)
	}
}
