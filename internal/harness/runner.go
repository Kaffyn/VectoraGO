package harness

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Kaffyn/Vectora/internal/core/engine"
	"github.com/Kaffyn/Vectora/internal/storage/db"
)

// TestCase define um caso de teste para o Vectora
type TestCase struct {
	ID          string            `yaml:"id"`
	Description string            `yaml:"description"`
	Input       string            `yaml:"input"`
	Memory      string            `yaml:"memory,omitempty"` // Contexto prévio injetado
	Expected    ExpectedResult    `yaml:"expected"`
	Config      TestConfig        `yaml:"config"`
}

// ExpectedResult define o que se espera da resposta
type ExpectedResult struct {
	Contains    []string `yaml:"contains"`    // Strings que devem estar na resposta
	NotContains []string `yaml:"not_contains"` // Strings que NÃO devem estar
	MinScore    float32  `yaml:"min_score"`    // Score mínimo de fidelidade (se avaliado por LLM)
	RequiredTools []string `yaml:"required_tools"` // Ferramentas que DEVEM ser chamadas
}

// TestConfig configurações específicas para o teste
type TestConfig struct {
	Timeout  time.Duration `yaml:"timeout"`
	Model    string        `yaml:"model"`
	TenantID string        `yaml:"tenant_id"`
}

// TestResult consolidado da execução
type TestResult struct {
	TestCaseID string
	Passed     bool
	Score      float32
	Duration   time.Duration
	Output     string
	Error      error
	ToolsUsed  []string
}

// Runner orquestra a execução dos testes
type Runner struct {
	engine *engine.Engine
}

func NewRunner(eng *engine.Engine) *Runner {
	return &Runner{engine: eng}
}

// Run executa um único caso de teste
func (r *Runner) Run(ctx context.Context, tc TestCase) (*TestResult, error) {
	start := time.Now()
	
	runCtx := ctx
	if tc.Config.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, tc.Config.Timeout)
		defer cancel()
	}

	// Workspace ID padrão para testes do harness
	wsID := tc.Config.TenantID
	if wsID == "" {
		wsID = "harness_test"
	}

	// Nota: No harness, usamos StreamQuery para capturar tokens e ferramentas em tempo real
	respChan, err := r.engine.StreamQuery(runCtx, tc.Input, wsID, tc.Config.Model, "chat", "en")
	if err != nil {
		return nil, err
	}
	
	var finalOutput strings.Builder
	var toolsUsed []string
	var seenTools = make(map[string]bool)
	
	for chunk := range respChan {
		if chunk.Token != "" {
			finalOutput.WriteString(chunk.Token)
		}
		if len(chunk.ToolCalls) > 0 {
			for _, t := range chunk.ToolCalls {
				if !seenTools[t] {
					toolsUsed = append(toolsUsed, t)
					seenTools[t] = true
				}
			}
		}
	}

	output := finalOutput.String()
	result := &TestResult{
		TestCaseID: tc.ID,
		Output:     output,
		Duration:   time.Since(start),
		ToolsUsed:  toolsUsed,
		Passed:     true,
	}

	// 1. Validar Contains
	for _, s := range tc.Expected.Contains {
		if !strings.Contains(strings.ToLower(output), strings.ToLower(s)) {
			result.Passed = false
			result.Error = fmt.Errorf("output missing expected string: %s", s)
			return result, nil
		}
	}

	// 2. Validar NotContains
	for _, s := range tc.Expected.NotContains {
		if strings.Contains(strings.ToLower(output), strings.ToLower(s)) {
			result.Passed = false
			result.Error = fmt.Errorf("output contains forbidden string: %s", s)
			return result, nil
		}
	}

	// 3. Validar Ferramentas Obrigatórias
	for _, reqTool := range tc.Expected.RequiredTools {
		if !seenTools[reqTool] {
			result.Passed = false
			result.Error = fmt.Errorf("required tool not called: %s", reqTool)
			return result, nil
		}
	}

	// 4. Validar Score (Mocked for now)
	if tc.Expected.MinScore > 0 {
		result.Score = 0.9 // Placeholder
		if result.Score < tc.Expected.MinScore {
			result.Passed = false
			result.Error = fmt.Errorf("quality score %.2f below minimum %.2f", result.Score, tc.Expected.MinScore)
		}
	}

	return result, nil
}
