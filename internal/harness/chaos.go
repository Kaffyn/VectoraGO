package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/Kaffyn/Vectora/internal/tools"
)

// FaultType represents the kind of fault to inject.
type FaultType string

const (
	FaultTypeError   FaultType = "error"
	FaultTypeTimeout FaultType = "timeout"
	FaultTypePartial FaultType = "partial" // returns truncated/incomplete output
)

// ChaosWrapper wraps a real tool and injects faults based on the FaultInjection config.
// It is the "Chaos Monkey" described in the Sprint plan.
type ChaosWrapper struct {
	inner    tools.Tool
	config   FaultInjection
	rng      *rand.Rand
	injected int // count of injections triggered
}

// NewChaosWrapper creates a fault-injecting wrapper around a real tool.
func NewChaosWrapper(inner tools.Tool, config FaultInjection) *ChaosWrapper {
	return &ChaosWrapper{
		inner:  inner,
		config: config,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (c *ChaosWrapper) Name() string        { return c.inner.Name() }
func (c *ChaosWrapper) Description() string  { return c.inner.Description() }
func (c *ChaosWrapper) Schema() json.RawMessage { return c.inner.Schema() }

// Execute intercepts the tool call and potentially injects a fault.
func (c *ChaosWrapper) Execute(ctx context.Context, args json.RawMessage) (*tools.ToolResult, error) {
	if !c.config.Enabled {
		return c.inner.Execute(ctx, args)
	}

	// Check if this tool is a fault injection target
	isTarget := false
	for _, t := range c.config.TargetTools {
		if t == c.inner.Name() || t == "*" {
			isTarget = true
			break
		}
	}

	if isTarget && c.rng.Float32() < c.config.FailureRate {
		c.injected++
		return c.injectFault(ctx)
	}

	return c.inner.Execute(ctx, args)
}

// injectFault creates the fault based on the configured failure type.
func (c *ChaosWrapper) injectFault(ctx context.Context) (*tools.ToolResult, error) {
	switch FaultType(c.config.FailureType) {
	case FaultTypeTimeout:
		timeoutMs := c.config.TimeoutMs
		if timeoutMs <= 0 {
			timeoutMs = 5000 // default 5s simulated timeout
		}
		select {
		case <-ctx.Done():
			return &tools.ToolResult{
				Output:  "context cancelled during fault injection",
				IsError: true,
			}, ctx.Err()
		case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
			return &tools.ToolResult{
				Output:  fmt.Sprintf("[CHAOS] Tool '%s' timed out after %dms (fault injection)", c.inner.Name(), timeoutMs),
				IsError: true,
			}, nil
		}

	case FaultTypePartial:
		// Actually execute but return only partial result
		result, err := c.inner.Execute(ctx, nil)
		if err != nil {
			return result, err
		}
		// Truncate to 20% of real output
		cutoff := len(result.Output) / 5
		if cutoff < 10 {
			cutoff = 10
		}
		result.Output = result.Output[:cutoff] + "\n[CHAOS] Partial output — simulated network fragmentation"
		return result, nil

	default: // FaultTypeError
		return &tools.ToolResult{
			Output:  fmt.Sprintf("[CHAOS] Tool '%s' failed: Service Unavailable (503) — fault injection", c.inner.Name()),
			IsError: true,
		}, nil
	}
}

// InjectionCount returns how many faults have been injected so far.
func (c *ChaosWrapper) InjectionCount() int {
	return c.injected
}

// WrapRegistry wraps all target tools in a Registry with Chaos wrappers.
// Returns a map from tool name to its ChaosWrapper for metrics collection.
func WrapRegistry(toolMap map[string]tools.Tool, config FaultInjection) map[string]*ChaosWrapper {
	wrappers := make(map[string]*ChaosWrapper)
	if !config.Enabled {
		return wrappers
	}

	for name, tool := range toolMap {
		shouldWrap := false
		for _, target := range config.TargetTools {
			if target == name || target == "*" {
				shouldWrap = true
				break
			}
		}
		if shouldWrap {
			wrappers[name] = NewChaosWrapper(tool, config)
		}
	}
	return wrappers
}
