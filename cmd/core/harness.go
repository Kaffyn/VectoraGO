package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kaffyn/Vectora/internal/harness"
	"github.com/Kaffyn/Vectora/internal/storage/infra"
	"github.com/Kaffyn/Vectora/internal/core/engine"
	"github.com/Kaffyn/Vectora/internal/storage/db"
	"github.com/Kaffyn/Vectora/internal/llm"
	"github.com/Kaffyn/Vectora/internal/tools"
	"github.com/Kaffyn/Vectora/internal/config/policies"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	harnessDir   string
	harnessModel string
)

var harnessCmd = &cobra.Command{
	Use:   "harness run [path]",
	Short: "Run quality harness tests",
	Long:  "Execute a suite of automated quality tests defined in YAML files to evaluate Vectora's performance.",
	RunE: func(cmd *cobra.Command, args []string) error {
		testDir := harnessDir
		if len(args) > 0 {
			testDir = args[0]
		}
		return runHarness(testDir)
	},
}

func runHarness(testDir string) error {
	fmt.Printf("🚀 Starting Vectora Harness Runner...\n")
	
	// 1. Setup minimal engine for tests
	ctx := context.Background()
	cfg := infra.LoadConfig()
	prefs := infra.LoadPreferences()
	llmRouter := llm.SetupRouter(ctx, cfg, prefs)
	
	// Utilizar workspace temporário para testes
	tempDir, _ := os.MkdirTemp("", "vectora-harness-*")
	defer os.RemoveAll(tempDir)
	
	kvStore, _ := db.NewKVStoreAtPath(filepath.Join(tempDir, "kv.db"))
	vecStore, _ := db.NewVectorStoreAtPath(filepath.Join(tempDir, "vec.db"))
	
	guardian := policies.NewGuardian(tempDir)
	toolsReg := tools.NewRegistry(tempDir, guardian, kvStore)
	
	eng := engine.NewEngine(vecStore, kvStore, llmRouter, toolsReg, guardian, nil)
	
	judge := harness.NewJudger(llmRouter, harness.JudgeConfig{Method: "llm_as_a_judge"})
	runner := harness.NewRunner(eng, judge)
	
	// 2. Load tests recursively
	var testCases []harness.TestCase
	err = filepath.WalkDir(testDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(d.Name(), ".yaml") || strings.HasSuffix(d.Name(), ".yml")) {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			var tc harness.TestCase
			if err := yaml.Unmarshal(data, &tc); err == nil {
				if tc.ID != "" {
					testCases = append(testCases, tc)
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk harness directory: %w", err)
	}
	
	if len(testCases) == 0 {
		fmt.Println("No test cases found in", testDir)
		return nil
	}
	
	fmt.Printf("Loaded %d test cases. Running...\n\n", len(testCases))
	
	passed := 0
	for _, tc := range testCases {
		fmt.Printf("TEST [%s]: %s\n", tc.ID, tc.Description)
		
		// Override model if specified in flag
		if harnessModel != "" {
			tc.Config.Model = harnessModel
		}
		
		res, err := runner.Run(ctx, tc)
		if err != nil {
			fmt.Printf("  ❌ ERROR: %v\n", err)
			continue
		}
		
		if res.Passed {
			passed++
			fmt.Printf("  ✅ PASSED (%.2fs)\n", res.Duration.Seconds())
			if res.JudgeVerdict != nil {
				fmt.Printf("      Judge Score: %.2f | Pass: %v\n", res.JudgeVerdict.Score, res.JudgeVerdict.PassThreshold)
				for dim, s := range res.JudgeVerdict.Dimensions {
					fmt.Printf("      - %s: %.2f\n", dim, s)
				}
			}
		} else {
			fmt.Printf("  ❌ FAILED: %v\n", res.Error)
			if res.JudgeVerdict != nil {
				fmt.Printf("      Judge Reasoning: %s\n", res.JudgeVerdict.Reasoning)
			}
			fmt.Printf("  --- Output Snippet ---\n%s\n----------------------\n", truncate(res.Output, 200))
		}
	}
	
	fmt.Printf("\n--- HARNESS SUMMARY ---\n")
	fmt.Printf("Total: %d | Passed: %d | Failed: %d\n", len(testCases), passed, len(testCases)-passed)
	
	if passed < len(testCases) {
		os.Exit(1)
	}
	
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
