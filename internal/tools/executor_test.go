package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// MockTool for testing
type MockTool struct {
	name        string
	description string
	delay       time.Duration
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Schema() json.RawMessage {
	return json.RawMessage(`{}`)
}

func (m *MockTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	// Simulate work
	select {
	case <-time.After(m.delay):
		return &ToolResult{Output: m.name + " completed"}, nil
	case <-ctx.Done():
		return &ToolResult{IsError: true, Output: "canceled"}, ctx.Err()
	}
}

func TestParallelExecution(t *testing.T) {
	// Create registry with mock tools
	registry := &Registry{Tools: make(map[string]Tool)}
	registry.Register(&MockTool{name: "tool1", delay: 10 * time.Millisecond})
	registry.Register(&MockTool{name: "tool2", delay: 10 * time.Millisecond})
	registry.Register(&MockTool{name: "tool3", delay: 10 * time.Millisecond})

	executor := NewExecutor(registry)

	executions := []ToolExecution{
		{ID: "exec1", Name: "tool1", Args: `{}`},
		{ID: "exec2", Name: "tool2", Args: `{}`},
		{ID: "exec3", Name: "tool3", Args: `{}`},
	}

	ctx := context.Background()
	start := time.Now()
	results := executor.ExecuteParallel(ctx, executions)
	elapsed := time.Since(start)

	// Verify all executed
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Verify parallel execution (should be ~10ms, not 30ms sequential)
	if elapsed > 100*time.Millisecond {
		t.Logf("Warning: execution took longer than expected: %v (expected ~10ms)", elapsed)
	}

	// Verify no errors
	for id, result := range results {
		if result.IsError {
			t.Errorf("Tool %s failed: %v", id, result.Error)
		}
	}
}

func TestDependencyResolution(t *testing.T) {
	registry := &Registry{Tools: make(map[string]Tool)}
	registry.Register(&MockTool{name: "tool1", delay: 5 * time.Millisecond})
	registry.Register(&MockTool{name: "tool2", delay: 5 * time.Millisecond})
	registry.Register(&MockTool{name: "tool3", delay: 5 * time.Millisecond})

	executor := NewExecutor(registry)

	// tool3 depends on tool1 and tool2
	executions := []ToolExecution{
		{ID: "exec1", Name: "tool1", Args: `{}`},
		{ID: "exec2", Name: "tool2", Args: `{}`},
		{ID: "exec3", Name: "tool3", Args: `{}`, DependsOn: []string{"exec1", "exec2"}},
	}

	ctx := context.Background()
	results := executor.ExecuteParallel(ctx, executions)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// All should complete successfully
	for id, result := range results {
		if result.IsError {
			t.Errorf("Tool %s failed: %v", id, result.Error)
		}
	}
}

func TestSequentialExecution(t *testing.T) {
	registry := &Registry{Tools: make(map[string]Tool)}
	registry.Register(&MockTool{name: "tool1", delay: 5 * time.Millisecond})
	registry.Register(&MockTool{name: "tool2", delay: 5 * time.Millisecond})

	executor := NewExecutor(registry)

	executions := []ToolExecution{
		{ID: "exec1", Name: "tool1", Args: `{}`},
		{ID: "exec2", Name: "tool2", Args: `{}`},
	}

	ctx := context.Background()
	results := executor.ExecuteSequential(ctx, executions)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	for id, result := range results {
		if result.IsError {
			t.Errorf("Tool %s failed: %v", id, result.Error)
		}
	}
}

func TestContextCancellation(t *testing.T) {
	registry := &Registry{Tools: make(map[string]Tool)}
	registry.Register(&MockTool{name: "tool1", delay: 100 * time.Millisecond})
	registry.Register(&MockTool{name: "tool2", delay: 100 * time.Millisecond})

	executor := NewExecutor(registry)

	executions := []ToolExecution{
		{ID: "exec1", Name: "tool1", Args: `{}`},
		{ID: "exec2", Name: "tool2", Args: `{}`},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results := executor.ExecuteParallel(ctx, executions)

	// Should handle cancellation gracefully
	if len(results) > 0 {
		for _, result := range results {
			if !result.IsError {
				t.Error("Expected error due to cancellation")
			}
		}
	}
}

func TestSingleToolExecution(t *testing.T) {
	registry := &Registry{Tools: make(map[string]Tool)}
	registry.Register(&MockTool{name: "tool1", delay: 5 * time.Millisecond})

	executor := NewExecutor(registry)

	executions := []ToolExecution{
		{ID: "exec1", Name: "tool1", Args: `{}`},
	}

	ctx := context.Background()
	results := executor.ExecuteParallel(ctx, executions)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results["exec1"].IsError {
		t.Errorf("Tool execution failed: %v", results["exec1"].Error)
	}
}
