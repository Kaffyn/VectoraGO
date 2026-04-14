package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ToolExecution represents a single tool call to be executed
type ToolExecution struct {
	ID       string
	Name     string
	Args     string
	DependsOn []string // IDs of tool calls this depends on
}

// ToolExecutionResult represents the result of a tool execution
type ToolExecutionResult struct {
	ID       string
	Output   string
	IsError  bool
	Error    error
	Metadata map[string]interface{}
}

// Executor provides concurrent tool execution with dependency resolution
type Executor struct {
	registry *Registry
	maxWorkers int
}

// NewExecutor creates a new concurrent tool executor
func NewExecutor(registry *Registry) *Executor {
	return &Executor{
		registry: registry,
		maxWorkers: 4, // Can be configured
	}
}

// ExecuteParallel executes multiple tool calls concurrently, respecting dependencies
// Returns a map of tool ID -> result
func (e *Executor) ExecuteParallel(ctx context.Context, executions []ToolExecution) map[string]*ToolExecutionResult {
	if len(executions) == 0 {
		return make(map[string]*ToolExecutionResult)
	}

	// Single tool: execute directly
	if len(executions) == 1 {
		result := e.executeSingle(ctx, executions[0])
		return map[string]*ToolExecutionResult{
			result.ID: result,
		}
	}

	// Multiple tools: use worker pool with dependency tracking
	results := make(map[string]*ToolExecutionResult)
	resultsMutex := sync.Mutex{}
	resultQueue := make(chan *ToolExecutionResult, len(executions))

	// Build dependency graph
	depGraph := buildDependencyGraph(executions)
	ready := getReadyExecutions(executions, depGraph, results)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, e.maxWorkers) // Limit concurrent executions

	// Result collection goroutine
	go func() {
		for result := range resultQueue {
			resultsMutex.Lock()
			results[result.ID] = result
			resultsMutex.Unlock()

			// Check if new executions are now ready
			newReady := getReadyExecutions(executions, depGraph, results)
			for _, exec := range newReady {
				wg.Add(1)
				go func(execution ToolExecution) {
					defer wg.Done()
					semaphore <- struct{}{}        // Acquire semaphore
					defer func() { <-semaphore }() // Release semaphore
					resultQueue <- e.executeSingle(ctx, execution)
				}(exec)
			}
		}
	}()

	// Initial worker pool for tools with no dependencies
	for _, exec := range ready {
		wg.Add(1)
		go func(execution ToolExecution) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore
			resultQueue <- e.executeSingle(ctx, execution)
		}(exec)
	}

	// Wait for all workers to complete and close result queue
	go func() {
		wg.Wait()
		close(resultQueue)
	}()

	// Drain result queue (already collecting in the goroutine above)
	for range resultQueue {
	}

	return results
}

// executeSingle executes a single tool call
func (e *Executor) executeSingle(ctx context.Context, exec ToolExecution) *ToolExecutionResult {
	result := &ToolExecutionResult{
		ID:       exec.ID,
		Metadata: make(map[string]interface{}),
	}

	tool, exists := e.registry.GetTool(exec.Name)
	if !exists {
		result.IsError = true
		result.Error = fmt.Errorf("tool '%s' not found", exec.Name)
		result.Output = result.Error.Error()
		return result
	}

	toolResult, err := tool.Execute(ctx, json.RawMessage(exec.Args))
	if err != nil {
		result.IsError = true
		result.Error = err
		result.Output = err.Error()
		return result
	}

	if toolResult != nil {
		result.IsError = toolResult.IsError
		result.Output = toolResult.Output
		result.Metadata = toolResult.Metadata
	}

	return result
}

// buildDependencyGraph creates a map of tool ID -> its dependencies
func buildDependencyGraph(executions []ToolExecution) map[string][]string {
	graph := make(map[string][]string)
	for _, exec := range executions {
		graph[exec.ID] = exec.DependsOn
	}
	return graph
}

// getReadyExecutions returns tools that are ready to execute
// (all their dependencies have completed successfully)
func getReadyExecutions(executions []ToolExecution, depGraph map[string][]string, completedResults map[string]*ToolExecutionResult) []ToolExecution {
	var ready []ToolExecution

	for _, exec := range executions {
		// Skip if already completed
		if _, done := completedResults[exec.ID]; done {
			continue
		}

		// Check if all dependencies are met
		allDepsMet := true
		for _, depID := range depGraph[exec.ID] {
			depResult, completed := completedResults[depID]
			if !completed || depResult.IsError {
				allDepsMet = false
				break
			}
		}

		if allDepsMet {
			ready = append(ready, exec)
		}
	}

	return ready
}

// ExecuteSequential executes tools in order (for compatibility)
func (e *Executor) ExecuteSequential(ctx context.Context, executions []ToolExecution) map[string]*ToolExecutionResult {
	results := make(map[string]*ToolExecutionResult)

	for _, exec := range executions {
		select {
		case <-ctx.Done():
			results[exec.ID] = &ToolExecutionResult{
				ID:      exec.ID,
				IsError: true,
				Error:   ctx.Err(),
				Output:  ctx.Err().Error(),
			}
			return results
		default:
		}

		result := e.executeSingle(ctx, exec)
		results[exec.ID] = result

		// Stop on error if tool failed
		if result.IsError {
			return results
		}
	}

	return results
}
