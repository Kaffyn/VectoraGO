package evaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Kaffyn/Vectora/internal/harness"
	"github.com/Kaffyn/Vectora/internal/llm"
)

// codeRepairRubric is the default rubric used by the LLM Judge.
// Each dimension is independently scored 0.0-1.0 by the Judge LLM.
const codeRepairRubric = `You are a senior software engineer performing a code review.
Evaluate the agent's output against the following criteria:

1. **Correctness** (0.0-1.0): Does the generated code actually solve the stated problem?
2. **Maintainability** (0.0-1.0): Is the code clean, readable, and following the project's conventions?
3. **Performance** (0.0-1.0): Are there obvious inefficiencies or memory leaks?
4. **Security** (0.0-1.0): Does the change introduce vulnerabilities or leave security gaps?
5. **Side Effects** (0.0-1.0): Does the agent ONLY modify what was requested, or does it touch unrelated files?

Return a JSON object with this exact structure:
{
  "score": <weighted_average_0_to_1>,
  "dimensions": {
    "correctness": <0_to_1>,
    "maintainability": <0_to_1>,
    "performance": <0_to_1>,
    "security": <0_to_1>,
    "side_effects": <0_to_1>
  },
  "reasoning": "<brief_explanation>",
  "pass_threshold": <true_if_score_gte_0.75>,
  "recommendations": ["<improvement1>", "<improvement2>"]
}`

// Judger uses an LLM to evaluate the quality of agent output.
// It is the "LLM-as-a-Judge" implementation for the Vectora Harness.
type Judger struct {
	llmRouter *llm.Router
	config    harness.JudgeConfig
}

// NewJudger creates a new Judger instance.
func NewJudger(router *llm.Router, config harness.JudgeConfig) *Judger {
	return &Judger{
		llmRouter: router,
		config:    config,
	}
}

// Evaluate runs the LLM Judge on the agent's output and returns a JudgeVerdict.
// task describes what was asked, agentOutput is the full agent response.
func (j *Judger) Evaluate(ctx context.Context, task, agentOutput string) (*harness.JudgeVerdict, error) {
	if j.config.Method == "deterministic" || j.config.Method == "" {
		// Fallback: simple length + pattern heuristic (no LLM cost)
		return j.deterministicEval(agentOutput), nil
	}

	rubric := codeRepairRubric
	if j.config.RubricPath != "" {
		// In a full implementation, load from file:
		// rubric, _ = os.ReadFile(j.config.RubricPath)
	}

	judgePrompt := fmt.Sprintf("%s\n\n---\n## TASK\n%s\n\n## AGENT OUTPUT\n%s", rubric, task, agentOutput)

	provider := j.llmRouter.GetDefault()
	if provider == nil || !provider.IsConfigured() {
		return nil, fmt.Errorf("no LLM provider available for Judge")
	}

	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a strict code quality judge. Always respond with valid JSON only."},
			{Role: llm.RoleUser, Content: judgePrompt},
		},
		Model:       j.config.Model,
		MaxTokens:   512,
		Temperature: 0.1, // near-zero for deterministic evaluation
	})
	if err != nil {
		return nil, fmt.Errorf("judge LLM call failed: %w", err)
	}

	return j.parseVerdict(resp.Content)
}

// parseVerdict extracts a JudgeVerdict from the LLM response.
func (j *Judger) parseVerdict(raw string) (*harness.JudgeVerdict, error) {
	// Strip markdown code fences if present
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var verdict harness.JudgeVerdict
	if err := json.Unmarshal([]byte(raw), &verdict); err != nil {
		return nil, fmt.Errorf("failed to parse judge verdict JSON: %w\nRaw: %s", err, raw)
	}

	return &verdict, nil
}

// deterministicEval is a fast heuristic judge that doesn't consume LLM tokens.
// Used for P0 tests, pre-commit hooks, or when method == "deterministic".
func (j *Judger) deterministicEval(agentOutput string) *harness.JudgeVerdict {
	score := float32(0.7) // baseline "reasonable" score

	// Penalize if output contains suspicious patterns
	if strings.Contains(agentOutput, "TODO") || strings.Contains(agentOutput, "FIXME") {
		score -= 0.1
	}
	if strings.Contains(strings.ToLower(agentOutput), "panic(") {
		score -= 0.15
	}
	if strings.Contains(agentOutput, "unsafe.Pointer") {
		score -= 0.2
	}

	// Reward for best practices
	if strings.Contains(agentOutput, "context.Context") {
		score += 0.05
	}
	if strings.Contains(agentOutput, "fmt.Errorf") || strings.Contains(agentOutput, "errors.Is") {
		score += 0.05
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return &harness.JudgeVerdict{
		Score: score,
		Dimensions: map[string]float32{
			"deterministic_heuristics": score,
		},
		Reasoning:     "Deterministic lint-style check (no LLM). Upgrade to llm_as_a_judge for full rubric scoring.",
		PassThreshold: score >= 0.75,
	}
}
