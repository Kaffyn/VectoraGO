package harness

import (
	"time"
)

// APIVersion represents the schema version
const APIVersion = "harness.vectora.dev/v2"

// TestCase represents the full YAML v2 specification for a test case
type TestCase struct {
	APIVersion    string                 `yaml:"api_version"`
	ID            string                 `yaml:"id"`
	Name          string                 `yaml:"name"`
	Description   string                 `yaml:"description"`
	Tags          []string               `yaml:"tags"`
	Priority      string                 `yaml:"priority"`      // P0, P1, P2, P3
	Execution     ExecutionConfig        `yaml:"execution"`
	Task          TaskDefinition         `yaml:"task"`
	Expectations  Expectations           `yaml:"expectations"`
	Evaluation    EvaluationConfig       `yaml:"evaluation"`
	Observability ObservabilityConfig    `yaml:"observability"`
}

// ExecutionConfig defines how the test should be set up
type ExecutionConfig struct {
	Workspace WorkspaceConfig `yaml:"workspace"`
	Routing   ModelRouting    `yaml:"model_routing"`
	Safety    SafetyConfig    `yaml:"safety"`
}

type WorkspaceConfig struct {
	Path         string `yaml:"path"`
	IndexMode    string `yaml:"index_mode"` // semantic, structural, hybrid, graph-only
	PreIndex     bool   `yaml:"pre_index"`
}

type ModelRouting struct {
	Preferred        []string `yaml:"preferred"`
	Fallback         []string `yaml:"fallback"`
	CostBudgetTokens int      `yaml:"cost_budget_tokens"`
	LatencySLAMS     int      `yaml:"latency_sla_ms"`
}

type SafetyConfig struct {
	TrustFolderEnforced bool     `yaml:"trust_folder_enforced"`
	GitSnapshotRequired bool     `yaml:"git_snapshot_required"`
	ApprovalGateTools   []string `yaml:"approval_gate_tools"`
}

// TaskDefinition defines the prompt and intent
type TaskDefinition struct {
	Type           string `yaml:"type"` // code_repair, refactor, doc_gen, etc.
	Prompt         string `yaml:"prompt"`
	ExpectedIntent string `yaml:"expected_intent"`
}

// Expectations defines the assertions for retrieval, tooling and output
type Expectations struct {
	Retrieval RetrievalExpectations `yaml:"retrieval"`
	Tooling   ToolingExpectations   `yaml:"tooling"`
	Output    OutputExpectations    `yaml:"output"`
}

type RetrievalExpectations struct {
	RequiredChunks []ChunkExpectation `yaml:"required_chunks"`
	ForbiddenChunks []ChunkExpectation `yaml:"forbidden_chunks"`
}

type ChunkExpectation struct {
	Path              string  `yaml:"path"`
	MinRelevanceScore float32 `yaml:"min_relevance_score"`
	StructuralRole    string  `yaml:"structural_role"` // source_of_truth, pattern_reference
	Reason            string  `yaml:"reason"`          // for forbidden chunks
}

type ToolingExpectations struct {
	// StrictSequence: tools must be called in this exact order
	StrictSequence []ToolStepExpectation `yaml:"strict_sequence"`
	// AnyOrder: tools must ALL be called but order doesn't matter (non-deterministic agent creativity)
	AnyOrder       []string              `yaml:"any_order"`
	// ForbiddenTools: these tools must NEVER be called
	ForbiddenTools []string              `yaml:"forbidden_tools"`

	// Legacy field for backward compatibility
	RequiredSequence []ToolStepExpectation `yaml:"required_sequence"`
}

type ToolStepExpectation struct {
	Tool    string         `yaml:"tool"`
	Args    map[string]any `yaml:"args"`
	MaxStep int            `yaml:"max_step"`
}

type OutputExpectations struct {
	SemanticChecks   []SemanticCheck   `yaml:"semantic_checks"`
	StructuralChecks []StructuralCheck `yaml:"structural_checks"`
	SecurityChecks   SecurityChecks    `yaml:"security_checks"`
}

type SemanticCheck struct {
	Pattern       string  `yaml:"pattern"`
	Location      string  `yaml:"location"` // response_content, generated_code
	MinConfidence float32 `yaml:"min_confidence"`
	MustPreserve  bool    `yaml:"must_preserve"`
}

type StructuralCheck struct {
	Type     string `yaml:"type"` // go_build, go_test, rust_build, etc.
	Path     string `yaml:"path"`
	MustPass bool   `yaml:"must_pass"`
}

type SecurityChecks struct {
	GuardianApproved     bool `yaml:"guardian_approved"`
	NoSensitiveDataLeak bool `yaml:"no_sensitive_data_leak"`
}

// Constraints defines hard limits that must not be violated
type Constraints struct {
	MaxTokensTotal     int      `yaml:"max_tokens_total"`
	ForbiddenFiles     []string `yaml:"forbidden_files"`     // paths that must never be written
	MaxRetriesPerTool  int      `yaml:"max_retries_per_tool"` // prevents runaway retry loops
	MaxWallTimeSeconds int      `yaml:"max_wall_time_seconds"`
}

// CodeQualityAnalysis defines static analysis checks
type CodeQualityAnalysis struct {
	Tool         string `yaml:"tool"`          // e.g. "golangci-lint"
	MaxNewIssues int    `yaml:"max_new_issues"` // 0 = zero tolerance
}

// FaultInjection defines chaos monkey behavior for resilience testing
type FaultInjection struct {
	Enabled          bool     `yaml:"enabled"`
	TargetTools      []string `yaml:"target_tools"`      // which tools to inject failures into
	FailureRate      float32  `yaml:"failure_rate"`      // 0.0-1.0 probability of failure
	FailureType      string   `yaml:"failure_type"`      // "timeout", "error", "partial"
	TimeoutMs        int      `yaml:"timeout_ms"`        // for timeout injection
	ExpectRecovery   bool     `yaml:"expect_recovery"`   // agent must recover gracefully
}

// EvaluationConfig defines scoring rules and judging method
type EvaluationConfig struct {
	Scoring     ScoringConfig      `yaml:"scoring"`
	Thresholds  Thresholds         `yaml:"thresholds"`
	Judge       JudgeConfig        `yaml:"judge_config"`
	Constraints Constraints        `yaml:"constraints"`
	Analysis    CodeQualityAnalysis `yaml:"code_quality"`
	FaultInject FaultInjection     `yaml:"fault_injection"`
}

type ScoringConfig struct {
	RetrievalMRRWeight         float32 `yaml:"retrieval_mrr_weight"`
	ToolSequenceAccuracyWeight float32 `yaml:"tool_sequence_accuracy_weight"`
	OutputCorrectnessWeight    float32 `yaml:"output_correctness_weight"`
	SafetyComplianceWeight     float32 `yaml:"safety_compliance_weight"`
}

type Thresholds struct {
	OverallPassScore float32  `yaml:"overall_pass_score"`
	MandatoryChecks  []string `yaml:"mandatory_checks"`
}

type JudgeConfig struct {
	Method     string `yaml:"method"` // llm_as_a_judge, deterministic
	Model      string `yaml:"model"`
	RubricPath string `yaml:"rubric_path"`
}

// ObservabilityConfig sets debugging and reporting options
type ObservabilityConfig struct {
	TraceLevel      string   `yaml:"trace_level"` // minimal, standard, detailed, debug
	MetricsExport   []string `yaml:"metrics_export"`
	FailureArtifacts []string `yaml:"failure_artifacts"`
}

// TestResult represents the result of a harness run
type TestResult struct {
	TestCaseID string
	Passed     bool
	Score      float32
	Duration   time.Duration
	Output     string
	Error      error
	Metrics    ResultMetrics
	Trace      []TraceEntry
}

type ResultMetrics struct {
	MRR              float32
	Steps            int
	TokensUsed       int        // total tokens consumed in this run
	Cost             float64    // estimated cost in USD
	SafetyEvents     int        // number of Guardian blocks triggered
	FaultInjections  int        // number of injected failures
	RecoveredFaults  int        // how many injected failures the agent survived
	ContextPrecision float32    // chunks requested vs chunks actually referenced
	LatencyP95Ms     int64      // p95 latency per agent step in ms
}

type TraceEntry struct {
	Step     int
	Action   string  // "think", "tool_call", "observe", "fault_inject", "budget_exceeded"
	Details  string
	Severity string  // "info", "warn", "error", "critical"
	Tokens   int     // tokens used in this step
	LatencyMs int64  // latency of this specific step
}

// JudgeVerdict is the output of an LLM-as-a-Judge evaluation
type JudgeVerdict struct {
	Score           float32            `json:"score"`            // 0.0-1.0
	Dimensions      map[string]float32 `json:"dimensions"`       // rubric scores per dimension
	Reasoning       string             `json:"reasoning"`        // LLM explanation
	PassThreshold   bool               `json:"pass_threshold"`
	Recommendations []string           `json:"recommendations"` // improvements suggested
}
