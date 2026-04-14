package manager

import (
	"context"
	"testing"
	"time"
)

func TestCircuitBreakerInitialState(t *testing.T) {
	rp := NewResourcePool(ResourceConfig{})

	state := rp.GetCircuitBreakerState("tenant1")
	if state.State != CircuitClosed {
		t.Errorf("Initial state should be closed, got %s", state.State)
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	rp := NewResourcePool(ResourceConfig{})
	tenantID := "tenant1"

	// Record failures up to threshold
	for i := 0; i < FailureThreshold; i++ {
		rp.RecordLLMFailure(tenantID)
	}

	state := rp.GetCircuitBreakerState(tenantID)
	if state.State != CircuitOpen {
		t.Errorf("Circuit should be open after %d failures, got %s", FailureThreshold, state.State)
	}

	if state.FailureCount != FailureThreshold {
		t.Errorf("Failure count should be %d, got %d", FailureThreshold, state.FailureCount)
	}
}

func TestCircuitBreakerBlocksCallsWhenOpen(t *testing.T) {
	rp := NewResourcePool(ResourceConfig{})
	tenantID := "tenant1"

	// Open circuit
	for i := 0; i < FailureThreshold; i++ {
		rp.RecordLLMFailure(tenantID)
	}

	// Try to make call
	canCall, backoff, err := rp.CanMakeLLMCall(tenantID)
	if canCall {
		t.Error("Should not allow call when circuit is open")
	}
	if err == nil {
		t.Error("Should return error when circuit is open")
	}
	if backoff <= 0 {
		t.Error("Should return positive backoff duration")
	}
}

func TestCircuitBreakerExponentialBackoff(t *testing.T) {
	testCases := []struct {
		failureCount int
		minBackoff   time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
	}

	for _, tc := range testCases {
		backoff := calculateExponentialBackoff(tc.failureCount)
		if backoff < tc.minBackoff {
			t.Errorf("Backoff for %d failures should be >= %v, got %v", tc.failureCount, tc.minBackoff, backoff)
		}
	}
}

func TestCircuitBreakerMaxBackoff(t *testing.T) {
	// Very high failure count should cap at MaxBackoffDuration
	backoff := calculateExponentialBackoff(100)
	if backoff > MaxBackoffDuration {
		t.Errorf("Backoff should not exceed %v, got %v", MaxBackoffDuration, backoff)
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	rp := NewResourcePool(ResourceConfig{})
	tenantID := "tenant1"

	// Open circuit
	for i := 0; i < FailureThreshold; i++ {
		rp.RecordLLMFailure(tenantID)
	}

	// Simulate successful call in half-open state after timeout
	// Manually transition to half-open to simulate timeout passage
	rp.mu.Lock()
	rp.circuitBreakers[tenantID].State = CircuitHalfOpen
	rp.circuitBreakers[tenantID].OpenedAt = time.Now().Add(-CircuitResetTimeout - 1*time.Second)
	rp.mu.Unlock()

	// Record successes
	for i := 0; i < SuccessThresholdHalfOpen; i++ {
		rp.RecordLLMSuccess(tenantID)
	}

	state := rp.GetCircuitBreakerState(tenantID)
	if state.State != CircuitClosed {
		t.Errorf("Circuit should be closed after recovery, got %s", state.State)
	}
}

func TestCircuitBreakerReopensAfterHalfOpenFailure(t *testing.T) {
	rp := NewResourcePool(ResourceConfig{})
	tenantID := "tenant1"

	// Open circuit
	for i := 0; i < FailureThreshold; i++ {
		rp.RecordLLMFailure(tenantID)
	}

	// Transition to half-open
	rp.mu.Lock()
	rp.circuitBreakers[tenantID].State = CircuitHalfOpen
	rp.mu.Unlock()

	// Fail in half-open state
	rp.RecordLLMFailure(tenantID)

	state := rp.GetCircuitBreakerState(tenantID)
	if state.State != CircuitOpen {
		t.Errorf("Circuit should reopen after failure in half-open, got %s", state.State)
	}
}

func TestMultipleTenantCircuitBreakers(t *testing.T) {
	rp := NewResourcePool(ResourceConfig{})

	// Open circuit for tenant1
	for i := 0; i < FailureThreshold; i++ {
		rp.RecordLLMFailure("tenant1")
	}

	// Keep tenant2 circuit closed
	rp.RecordLLMSuccess("tenant2")

	state1 := rp.GetCircuitBreakerState("tenant1")
	state2 := rp.GetCircuitBreakerState("tenant2")

	if state1.State != CircuitOpen {
		t.Error("Tenant1 circuit should be open")
	}
	if state2.State != CircuitClosed {
		t.Error("Tenant2 circuit should be closed")
	}

	// Test calls
	can1, _, _ := rp.CanMakeLLMCall("tenant1")
	can2, _, _ := rp.CanMakeLLMCall("tenant2")

	if can1 {
		t.Error("Tenant1 should not be able to make call")
	}
	if !can2 {
		t.Error("Tenant2 should be able to make call")
	}
}

func TestAcquireLLMSlotWithCircuitBreaker(t *testing.T) {
	rp := NewResourcePool(ResourceConfig{
		MaxParallelLLMPerTenant: 2,
	})
	tenantID := "tenant1"

	// Should succeed when circuit is closed
	ctx := context.Background()
	err := rp.AcquireLLMSlot(tenantID, ctx)
	if err != nil {
		t.Errorf("Should acquire slot when circuit is closed: %v", err)
	}
	defer rp.ReleaseLLMSlot(tenantID)

	// Record success
	rp.RecordLLMSuccess(tenantID)

	state := rp.GetCircuitBreakerState(tenantID)
	if state.LastSuccess.IsZero() {
		t.Error("Last success should be recorded")
	}
}

func TestFailureCounterDecrementsOnSuccess(t *testing.T) {
	rp := NewResourcePool(ResourceConfig{})
	tenantID := "tenant1"

	// Record some failures
	for i := 0; i < 3; i++ {
		rp.RecordLLMFailure(tenantID)
	}

	state := rp.GetCircuitBreakerState(tenantID)
	if state.FailureCount != 3 {
		t.Errorf("Expected 3 failures, got %d", state.FailureCount)
	}

	// Record success
	rp.RecordLLMSuccess(tenantID)

	state = rp.GetCircuitBreakerState(tenantID)
	if state.FailureCount != 2 {
		t.Errorf("Failure count should decrement on success, got %d", state.FailureCount)
	}
}
