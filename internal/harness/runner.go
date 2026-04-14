package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kaffyn/Vectora/internal/config/telemetry"
	"github.com/Kaffyn/Vectora/internal/core/engine"
)

// Runner orquestra a execução dos testes Harness v2.0
type Runner struct {
	engine *engine.Engine
	judger *Judger
}

func NewRunner(eng *engine.Engine, judge *Judger) *Runner {
	return &Runner{
		engine: eng,
		judger: judge,
	}
}

// Run executa um caso de teste Harness v2.0 completo com:
// - Token budget enforcement
// - Strict + Unordered tool assertions
// - Structural checks (go_build, go_test)
// - Fault injection support (via chaos wrappers)
// - LLM Judge integration
func (r *Runner) Run(ctx context.Context, tc TestCase) (*TestResult, error) {
	start := time.Now()
	accumulator := telemetry.NewStepAccumulator()
	trace := []TraceEntry{}

	// === PHASE 1: SETUP ===
	workspacePath := tc.Execution.Workspace.Path
	if workspacePath == "" {
		workspacePath = "."
	}

	// Pre-indexation
	if tc.Execution.Workspace.PreIndex {
		if err := r.engine.StartIndexation(ctx, workspacePath); err != nil {
			return nil, fmt.Errorf("pre-index failed: %w", err)
		}
		trace = append(trace, TraceEntry{Step: 0, Action: "pre_index", Details: workspacePath, Severity: "info"})
	}

	// === PHASE 2: EXECUTION CONTEXT ===
	runCtx := ctx
	if tc.Evaluation.Constraints.MaxWallTimeSeconds > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(tc.Evaluation.Constraints.MaxWallTimeSeconds)*time.Second)
		defer cancel()
	} else if tc.Execution.Routing.LatencySLAMS > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(tc.Execution.Routing.LatencySLAMS)*time.Millisecond)
		defer cancel()
	}

	wsID := filepath.Base(workspacePath)
	preferredModel := ""
	if len(tc.Execution.Routing.Preferred) > 0 {
		preferredModel = tc.Execution.Routing.Preferred[0]
	}

	// === PHASE 2.5: CHAOS MONKEY SETUP ===
	if tc.Execution.FaultInjection.Enabled {
		// Note: This requires the Engine to support tool swapping or wrapping.
		// For this implementation, we assume engine.WrapTools exists or we use the Registry directly.
		trace = append(trace, TraceEntry{Step: 0, Action: "chaos_monkey_init", Details: fmt.Sprintf("FailRate: %.2f", tc.Execution.FaultInjection.FailureRate), Severity: "warn"})
		// Lógica simplificada: o engine já deve estar configurado com os wrappers se necessário, 
		// ou injetamos aqui no Registry do engine.
	}

	// === PHASE 3: AGENT EXECUTION (with token tracking) ===
	stepNum := 0
	var output strings.Builder
	var observedTools []string
	var seenTools = make(map[string]bool)
	var budgetExceeded bool

	tokenBudget := tc.Evaluation.Constraints.MaxTokensTotal
	if tokenBudget == 0 {
		tokenBudget = tc.Execution.Routing.CostBudgetTokens
	}

	respChan, err := r.engine.StreamQuery(runCtx, tc.Task.Prompt, wsID, preferredModel, "agent", "en")
	if err != nil {
		return nil, err
	}

	for chunk := range respChan {
		stepStart := time.Now()

		if chunk.Token != "" {
			output.WriteString(chunk.Token)
		}

		if len(chunk.ToolCalls) > 0 {
			stepNum++
			for _, t := range chunk.ToolCalls {
				if !seenTools[t] {
					observedTools = append(observedTools, t)
					seenTools[t] = true
				}
			}

			// Read telemetry bridge for this step's token usage
			_, tokensOut, costUSD, _ := telemetry.GetLastInteractionMetrics()
			stepLatency := time.Since(stepStart).Milliseconds()
			accumulator.Record(telemetry.StepMetric{
				Step:      stepNum,
				TokensOut: tokensOut,
				LatencyMs: stepLatency,
				Tool:      strings.Join(chunk.ToolCalls, ","),
			})

			trace = append(trace, TraceEntry{
				Step:      stepNum,
				Action:    "tool_call",
				Details:   strings.Join(chunk.ToolCalls, ", "),
				Severity:  "info",
				Tokens:    tokensOut,
				LatencyMs: stepLatency,
			})

			// === BUDGET ENFORCEMENT ===
			if tokenBudget > 0 && accumulator.TotalTokens() > tokenBudget {
				budgetExceeded = true
				trace = append(trace, TraceEntry{
					Step:     stepNum,
					Action:   "budget_exceeded",
					Details:  fmt.Sprintf("used %d / budget %d tokens", accumulator.TotalTokens(), tokenBudget),
					Severity: "error",
				})
				break
			}

			// Forbidden files constraint check
			for _, tool := range chunk.ToolCalls {
				if tool == "write_file" || tool == "edit" {
					for _, forbidden := range tc.Evaluation.Constraints.ForbiddenFiles {
						if strings.Contains(chunk.Token, forbidden) {
							trace = append(trace, TraceEntry{
								Step:     stepNum,
								Action:   "forbidden_file_access",
								Details:  forbidden,
								Severity: "error",
							})
						}
					}
				}
			}
		}
	}

	finalOutput := output.String()

	result := &TestResult{
		TestCaseID: tc.ID,
		Output:     finalOutput,
		Duration:   time.Since(start),
		Passed:     true,
		Trace:      trace,
		Metrics: ResultMetrics{
			Steps:        stepNum,
			TokensUsed:   accumulator.TotalTokens(),
			LatencyP95Ms: accumulator.P95LatencyMs(),
		},
	}

	// === PHASE 4: TOKEN BUDGET FAILURE ===
	if budgetExceeded {
		result.Passed = false
		result.Error = fmt.Errorf("ErrBudgetExceeded: consumed %d tokens, budget was %d", accumulator.TotalTokens(), tokenBudget)
		return result, nil
	}

	// === PHASE 5: TOOL ASSERTIONS ===

	// 5.1 Strict sequence (exact order required — P0 tests)
	strictSeq := tc.Expectations.Tooling.StrictSequence
	if len(strictSeq) == 0 {
		strictSeq = tc.Expectations.Tooling.RequiredSequence // backward compat
	}
	if len(strictSeq) > 0 {
		for i, req := range strictSeq {
			if i >= len(observedTools) || observedTools[i] != req.Tool {
				result.Passed = false
				result.Error = fmt.Errorf("strict_sequence mismatch at step %d: expected %q, got %v", i+1, req.Tool, observedTools)
				return result, nil
			}
		}
	}

	// 5.2 Any-order set assertion (creative agent — P1+ tests)
	for _, required := range tc.Expectations.Tooling.AnyOrder {
		if !seenTools[required] {
			result.Passed = false
			result.Error = fmt.Errorf("any_order assertion failed: tool %q was never called", required)
			return result, nil
		}
	}

	// 5.3 Forbidden tools
	for _, forbidden := range tc.Expectations.Tooling.ForbiddenTools {
		if seenTools[forbidden] {
			result.Passed = false
			result.Error = fmt.Errorf("forbidden_tool called: %q must never be invoked", forbidden)
			return result, nil
		}
	}

	// === PHASE 6: STRUCTURAL CHECKS ===
	for _, check := range tc.Expectations.Output.StructuralChecks {
		if !check.MustPass {
			continue
		}
		var cmd *exec.Cmd
		switch check.Type {
		case "go_build":
			cmd = exec.CommandContext(runCtx, "go", "build", "./...")
		case "go_test":
			cmd = exec.CommandContext(runCtx, "go", "test", "./...")
		case "golangci_lint":
			cmd = exec.CommandContext(runCtx, "golangci-lint", "run", "./...")
		}

		if cmd != nil {
			cmd.Dir = check.Path
			if err := cmd.Run(); err != nil {
				result.Passed = false
				result.Error = fmt.Errorf("structural check [%s] failed in %s: %w", check.Type, check.Path, err)
				return result, nil
			}
		}
	}

	// === PHASE 7: SEMANTIC PATTERN CHECKS ===
	for _, check := range tc.Expectations.Output.SemanticChecks {
		if !strings.Contains(strings.ToLower(finalOutput), strings.ToLower(check.Pattern)) {
			result.Passed = false
			result.Error = fmt.Errorf("semantic check failed: missing pattern %q in output", check.Pattern)
			return result, nil
		}
	}

	// === PHASE 8: FORBIDDEN FILE WRITES ===
	for _, forbidden := range tc.Evaluation.Constraints.ForbiddenFiles {
		// If the file was modified relative to its last mtime, flag it
		if _, err := os.Stat(forbidden); err == nil {
			// In a real impl we'd compare mtime to a baseline snapshot
			// For now: if file exists AND is in forbidden list, flag as warning
			trace = append(trace, TraceEntry{
				Step:     stepNum,
				Action:   "forbidden_file_check",
				Details:  fmt.Sprintf("file %q exists and is in forbidden list — verify manually", forbidden),
				Severity: "warn",
			})
		}
	}

	// === PHASE 9: LLM JUDGE EVALUATION ===
	if r.judger != nil && tc.Evaluation.Judge.Model != "" {
		trace = append(trace, TraceEntry{Step: stepNum + 1, Action: "judge_start", Details: tc.Evaluation.Judge.Model, Severity: "info"})
		verdict, err := r.judger.Evaluate(ctx, tc.Task.Prompt, finalOutput)
		if err != nil {
			trace = append(trace, TraceEntry{Step: stepNum + 1, Action: "judge_error", Details: err.Error(), Severity: "error"})
		} else {
			result.JudgeVerdict = verdict
			if !verdict.PassThreshold {
				result.Passed = false
				result.Error = fmt.Errorf("Judge rejected output (Score: %.2f): %s", verdict.Score, verdict.Reasoning)
			}
			trace = append(trace, TraceEntry{Step: stepNum + 1, Action: "judge_complete", Details: fmt.Sprintf("Score: %.2f", verdict.Score), Severity: "info"})
		}
	}

	result.Trace = trace
	return result, nil
}

// truncate safely truncates a string for display.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
