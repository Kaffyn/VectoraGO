# Vectora Quality Harness

The Harness is our specialized validation framework for agentic AI. It ensures that every code change maintains or improves the assistant's accuracy and safety.

## 1. Running Tests

```bash
vectora harness run ./harness/suites
```

Options:

- `--dir`: Custom path to test cases.
- `--model`: Force all tests to run on a specific model (e.g. `claude-4.6-sonnet`).

## 2. YAML Specification (v2)

Each test case is a YAML file adhering to the `harness.vectora.dev/v2` schema.

```yaml
id: "security-jwt-validation"
name: "JWT Expiry Check"
task:
  prompt: "Check if the token in auth.go has an expiry."
expectations:
  tooling:
    strict_sequence:
      - tool: "read_file"
        args: { "path": "auth.go" }
  output:
    semantic_checks:
      - pattern: "expiration"
evaluation:
  judge_config:
    method: "llm_as_a_judge"
    model: "gemini-3.1-pro"
```

## 3. The LLM-as-a-Judge

The Judge evaluates output using a 5-dimension rubric:

- **Correctness**: Is the answer factually accurate?
- **Maintainability**: Is the proposed code clean?
- **Security**: Are there vulnerable patterns?
- **Performance**: Is the logic efficient?
- **Side Effects**: Does it break unrelated code?

## 4. Fault Injection (Chaos Monkey)

The Harness can simulate real-world failures:

- **Timeouts**: Simulate network lag.
- **Errors**: Simulate tool crashes.
- **Partial Outputs**: Simulate corrupted data.

The agent must show **recovery behavior** to pass resilience tests.

---

_QA Standards: April 2026_
