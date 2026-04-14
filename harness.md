# Vectora Harness — Especificação Técnica

> **Sistema de Validação de Inteligência Agêntica e Recuperação Semântica**
>
> _"Não basta o código compilar. A inteligência precisa ser precisa, segura e previsível."_

---

## 📋 Visão Geral

O **Vectora Harness** é a camada de garantia de qualidade que valida se o sistema de IA do Vectora entrega resultados corretos, seguros e eficientes para tarefas reais de engenharia de software.

Diferente de testes unitários tradicionais, o Harness opera em três dimensões:

| Dimensão           | O que valida                                      | Exemplo                                                      |
| ------------------ | ------------------------------------------------- | ------------------------------------------------------------ |
| **🔍 Recuperação** | Precisão do RAG híbrido (semântico + estrutural)  | "A busca por 'JWT' recuperou `jwt.go` antes de `auth.go`?"   |
| **🧠 Raciocínio**  | Coerência do loop ReAct e decisões do agente      | "O agente planejou ler antes de editar? Lidou com erro?"     |
| **⚡ Execução**    | Correção das ferramentas e segurança das mutações | "O `edit` gerou um patch válido? O snapshot Git foi criado?" |

---

## 🎯 Objetivos Estratégicos

```yaml
strategic_goals:
  - name: "Precision First"
    target: "MRR@5 > 0.92 para queries de codebase"
    rationale: "Desenvolvedores não podem perder tempo com contexto irrelevante"

  - name: "Agent Reliability"
    target: "95% das tarefas completas em ≤5 passos ReAct"
    rationale: "Latência cognitiva impacta diretamente a produtividade"

  - name: "Safety by Default"
    target: "0 falsos positivos em Guardian checks"
    rationale: "Segurança não é negociável"
```

---

## 📐 Esquema Harness (YAML v2)

Os casos de teste vivem em `harness/*.yaml` e seguem este schema estrito:

```yaml
api_version: "harness.vectora.dev/v2"
id: "auth-jwt-expiration-fix"
name: "Correção de Validação de Expiração JWT"
description: "Valida se o agente identifica a falta de validação de 'exp' em tokens JWT."

# Contexto de execução
execution:
  workspace:
    path: "./internal/auth"
    index_mode: "hybrid"
    pre_index: true

  safety:
    trust_folder_enforced: true
    git_snapshot_required: true
    approval_gate_tools: ["run_shell_command", "write_file"]

# Tarefa enviada ao agente
task:
  type: "code_repair"
  prompt: "O middleware em internal/auth/jwt.go não está validando o campo 'exp'."

# Expectativas observáveis (assertions)
expectations:
  retrieval:
    required_chunks:
      - path: "internal/auth/jwt.go"
        min_relevance_score: 0.88

  tooling:
    required_sequence:
      - tool: "read_file"
        args: { path: "internal/auth/jwt.go" }
      - tool: "edit"
        args: { path: "internal/auth/jwt.go" }

  output:
    structural_checks:
      - type: "go_build"
        path: "./internal/auth"
        must_pass: true

evaluation:
  scoring:
    retrieval_mrr_weight: 0.30
    output_correctness_weight: 0.40
    safety_compliance_weight: 0.30
```

---

## 🏗️ Arquitetura do Runner

O comando `vectora harness run` orquestra a execução em 6 fases:

1.  **Orchestration Layer**: Parse do YAML e resolução de roteamento de modelos.
2.  **Sandbox Provisioning**: Clone do workspace em `/tmp` e snapshot Git inicial.
3.  **Agent Execution**: Inicia o loop em `internal/core/agent.go` interceptando chamadas.
4.  **Real-time Monitoring**: Validação de permissões pelo Guardian e cálculo de MRR@K.
5.  **Evaluation Pipeline**: Checks determinísticos (build/tests) + LLM-as-a-judge.
6.  **Teardown**: Limpeza do sandbox e exportação de métricas (Prometheus/JSON).

---

## 📊 Métricas e Scoring

| Métrica                | Target | Por que importa                               |
| ---------------------- | ------ | --------------------------------------------- |
| **MRR@5**              | ≥ 0.92 | Desenvolvedor encontra o código certo rápido  |
| **Step Efficiency**    | ≥ 0.75 | Evita loops infinitos e custos desnecessários |
| **Tool Accuracy**      | ≥ 0.95 | Ferramentas mal invocadas quebram a execução  |
| **Guardian Precision** | 1.0    | Bloquear arquivo legítimo é inaceitável       |

---

## 🧠 Loop de Raciocínio (Agent Core)

### Máquina de Estados do Agente (`internal/core/agent.go`)

```go
type AgentState struct {
    History         []llm.Message
    Steps           int
    MaxSteps        int
    BudgetTokens    int
    UsedTokens      int
    SafetyFlags     SafetyState
}

func (a *Agent) RunLoop(ctx context.Context, task *Task) (*Result, error) {
    state := a.initState(task)

    for state.Steps < state.MaxSteps {
        // THINK
        thought, _ := a.llm.Generate(ctx, state.History, a.tools.Schema())
        state.History = append(state.History, thought.Message)

        // DECIDE
        if len(thought.ToolCalls) == 0 {
            return a.evaluateFinalAnswer(state)
        }

        // ACT (com interceptação)
        results, _ := a.toolExecutor.ExecuteBatch(ctx, thought.ToolCalls)

        // OBSERVE
        for _, res := range results {
            state.History = append(state.History, llm.Message{
                Role: "tool",
                ToolCallID: res.ID,
                Content: res.Output,
            })
        }
        state.Steps++
    }
    return nil, fmt.Errorf("max_steps_reached")
}
```

---

## 🧰 Biblioteca de Suítes Oficiais

- **`suite:codebase-navigation`**: Busca de símbolos e cadeias de dependências.
- **`suite:refactor-patterns`**: Extração de interfaces e aplicação de design patterns.
- **`suite:security-audit`**: Correção de SQL injection e detecção de secrets.
- **`suite:documentation-qa`**: RAG sobre documentação técnica específica de versão.

---

## 🔄 Integração com CI/CD

O Harness pode ser integrado diretamente no GitHub Actions ou como um hook de `pre-commit`:

```bash
# Hook de Pre-Commit
vectora harness run \
  --suite quick-validation \
  --scope $(git diff --cached --name-only) \
  --model qwen-3.6-turbo
```

---

## ⚠️ Considerações de Segurança (Sandbox)

1.  **Trust Folder**: O agente só opera dentro do diretório do projeto.
2.  **Snapshot Git**: Cada mutação é precedida de um snapshot atômico para rollback.
3.  **Approval Gate**: Ferramentas sensíveis (`run_shell_command`) exigem confirmação manual.
4.  **Token Limit**: Interrupção automática se o custo exceder o budget definido.

---

**Próximos Passos**: Implementar o validador de schema em `internal/harness/schema/validator.go` e a primeira suíte de validação rápida.
