package telemetry

import (
	"sync"
	"sync/atomic"
)

// InteractionMetrics holds the last LLM interaction metrics for consumption by the Harness runner.
// It is thread-safe via atomic operations and a RWMutex for the cost float.
type InteractionMetrics struct {
	TokensIn  int64
	TokensOut int64
	mu        sync.RWMutex
	CostUSD   float64
	Provider  string
}

var lastMetrics InteractionMetrics

// RecordInteraction is called by LogLLMInteraction to persist metrics for Harness consumption.
// This is the "cable" that connects telemetry → runner → ResultMetrics.
func RecordInteraction(provider string, tokensIn, tokensOut int, costUsd float64) {
	atomic.StoreInt64(&lastMetrics.TokensIn, int64(tokensIn))
	atomic.StoreInt64(&lastMetrics.TokensOut, int64(tokensOut))
	lastMetrics.mu.Lock()
	lastMetrics.CostUSD = costUsd
	lastMetrics.Provider = provider
	lastMetrics.mu.Unlock()
}

// GetLastInteractionMetrics reads the most recent LLM interaction metrics.
// Returns (tokensIn, tokensOut, costUSD, provider).
func GetLastInteractionMetrics() (int, int, float64, string) {
	in := int(atomic.LoadInt64(&lastMetrics.TokensIn))
	out := int(atomic.LoadInt64(&lastMetrics.TokensOut))
	lastMetrics.mu.RLock()
	cost := lastMetrics.CostUSD
	prov := lastMetrics.Provider
	lastMetrics.mu.RUnlock()
	return in, out, cost, prov
}

// StepMetric holds per-step token and latency data for trace assembly.
type StepMetric struct {
	Step      int
	TokensIn  int
	TokensOut int
	LatencyMs int64
	Tool      string
}

// StepAccumulator collects per-step metrics during a Harness run.
type StepAccumulator struct {
	mu      sync.Mutex
	steps   []StepMetric
	totalIn int
	totalOut int
}

func NewStepAccumulator() *StepAccumulator {
	return &StepAccumulator{}
}

// Record records a step's metrics.
func (a *StepAccumulator) Record(m StepMetric) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.steps = append(a.steps, m)
	a.totalIn += m.TokensIn
	a.totalOut += m.TokensOut
}

// TotalTokens returns the accumulated token count for the full run.
func (a *StepAccumulator) TotalTokens() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.totalIn + a.totalOut
}

// P95LatencyMs computes the p95 latency across all recorded steps.
func (a *StepAccumulator) P95LatencyMs() int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.steps) == 0 {
		return 0
	}
	// Simple sort-based p95
	lats := make([]int64, len(a.steps))
	for i, s := range a.steps {
		lats[i] = s.LatencyMs
	}
	// Insertion sort (n is small, < 20 steps typically)
	for i := 1; i < len(lats); i++ {
		for j := i; j > 0 && lats[j] < lats[j-1]; j-- {
			lats[j], lats[j-1] = lats[j-1], lats[j]
		}
	}
	idx := int(float64(len(lats)) * 0.95)
	if idx >= len(lats) {
		idx = len(lats) - 1
	}
	return lats[idx]
}
