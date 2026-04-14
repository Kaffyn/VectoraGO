# Sistema Harness — Infraestrutura de Testes e Qualidade

O **Harness** é o sistema de garantia de qualidade do Vectora. Ele não testa apenas se o código "roda" (testes unitários), mas se a **inteligência** e a **recuperação semântica** são precisas para tarefas de engenharia de software do mundo real.

## 1. Objetivos do Harness

- **Validar a Precisão Semântica**: Garantir que o motor HNSW + TurboQuant recupere os chunks corretos para perguntas complexas.
- **Verificar Cadeias de Ferramentas**: Garantir que o Agente invoque as ferramentas corretas (`read_file`, `edit`, `grep_search`) na ordem lógica para resolver uma tarefa.
- **Benchmarking de Modelos**: Comparar a performance de diferentes LLMs (Gemini vs Claude vs GPT) sobre a mesma codebase.
- **Prevenção de Regressão**: Garantir que melhorias no orquestrador não quebrem a capacidade de compreensão de sistemas legados.

---

## 2. O Formato Harness (YAML)

Os casos de teste são definidos em arquivos `.yaml` localizados em `~/.vectora/harness/` ou na raiz do projeto em `/harness/`.

```yaml
id: "refactor-auth-logic"
name: "Refatoração de Middleware de Autenticação"
description: "Verifica se o agente consegue encontrar o erro no JWT e sugerir o patch correto."

# O contexto inicial do workspace
context:
  workspace_path: "./internal/auth"
  index_required: true

# A tarefa enviada ao agente
task: "O middleware de JWT não está validando a expiração do token. Encontre o arquivo e corrija."

# Expectativas de Comportamento (Garantias)
expectations:
  # O agente deve obrigatoriamente ler este arquivo
  files_accessed:
    - "internal/auth/jwt.go"

  # Ferramentas que DEVEM ser chamadas
  required_tools:
    - "read_file"
    - "edit"

  # Conteúdo que deve estar presente na resposta final
  min_semantic_score: 0.85
  expected_output_patterns:
    - "WithExpirationCheck"
    - "jwt.ParseWithClaims"

# Critérios de Sucesso
assertions:
  - type: "exit_code"
    value: 0
  - type: "git_diff_check"
    description: "Verifica se o patch não quebrou o build"
```

---

## 3. Arquitetura do Runner

O comando `vectora harness run` orquestra a execução:

1. **Setup de Sandbox**: Cria um ambiente temporário (clone do workspace) para evitar efeitos colaterais.
2. **Preparação de Memória**: Carrega o banco vetorial se `index_required` for true.
3. **Execução do Agente**: Inicia o loop de pensamento do agente sobre a `task`.
4. **Monitoramento de Ferramentas**: Intercepta todas as chamadas de ferramentas e meta-dados de busca.
5. **Avaliação (Scoring)**: Compara a execução com as `expectations` usando um **LLM Judge** ou regras determinísticas.

---

## 4. Métricas de Qualidade

O Harness gera um relatório com as seguintes métricas:

| Métrica                      | O que mede                                                       |
| :--------------------------- | :--------------------------------------------------------------- |
| **Pass Rate**                | Porcentagem de asserções que passaram.                           |
| **Retrieval Accuracy (MRR)** | O quão alto nos resultados da busca estava o chunk necessário.   |
| **Tool Efficiency**          | Número de passos tomados vs o caminho ótimo definido no harness. |
| **Latência por Passo**       | Tempo gasto em raciocínio vs execução de ferramenta.             |
| **Cost (Tokens)**            | Custo total da tarefa em tokens de entrada/saída.                |

---

## 5. Biblioteca de Suítes Integradas

O Vectora vem com suítes pré-configuradas para validação rápida:

- **Suite `codebase-navigation`**: Testes de busca de símbolos e navegação de dependências.
- **Suite `refactor-patterns`**: Testes de aplicação de design patterns via `edit`.
- **Suite `documentation-qa`**: Testes de perguntas e respostas sobre READMEs e docs técnicas.

---

## 6. Como Contribuir

Para adicionar um novo teste:

1. Crie um arquivo em `internal/harness/suites/`.
2. Defina a tarefa e as expectativas.
3. Rode `go test ./internal/harness/...` para validar o próprio harness.

---

# 🧠 Especificação: O Loop de Raciocínio (Agent Core)

Para que o Vectora execute o que foi definido no **Harness**, ele precisa de um motor que não apenas responda, mas que **decida**. Implementamos o padrão **ReAct (Reason + Act)** otimizado para a infraestrutura de baixa latência do Go.

## 1. O Ciclo de Vida da Requisição (The "Think" Loop)

O loop de raciocínio reside em `internal/core/agent.go`. Ele opera como uma máquina de estados finitos que itera até que o objetivo seja alcançado ou um limite de segurança seja atingido.

### As 4 Fases do Loop:

1. **Input Analysis**: O `intent.go` determina se a tarefa exige ação (Agente) ou apenas resposta (Chat).
2. **Thought (Raciocínio)**: A LLM emite um bloco de texto explicando sua estratégia.
   - _Ex: "Preciso ler o arquivo X para entender como o JWT é validado."_
3. **Action (Ação)**: A LLM invoca uma ferramenta do `internal/tools`. O Vectora intercepta isso, valida as permissões e executa.
4. **Observation (Observação)**: O output da ferramenta (sucesso, erro ou conteúdo) é anexado ao histórico como uma nova entrada de sistema, alimentando a próxima iteração.

---

## 2. Implementação Técnica: `internal/core/agent.go`

Abaixo, a estrutura de código para suportar o loop iterativo com suporte a cancelamento via `Context`.

```go
type AgentState struct {
    History []llm.Message
    Steps   int
    MaxSteps int
}

func (v *Vectora) RunLoop(ctx context.Context, task string) error {
    state := &AgentState{
        MaxSteps: 10,
        History: []llm.Message{
            {Role: "system", Content: v.generateSystemPrompt()},
            {Role: "user", Content: task},
        },
    }

    for state.Steps < state.MaxSteps {
        // 1. THINK: Chama a LLM com as ferramentas disponíveis
        resp, err := v.llmMgr.Generate(ctx, state.History, v.tools.List())
        if err != nil {
            return err
        }

        // 2. PROCESS: Verifica se há chamadas de ferramentas
        if len(resp.ToolCalls) == 0 {
            v.telemetry.LogFinalAnswer(resp.Content)
            return nil // Objetivo alcançado
        }

        // 3. ACT: Executa ferramentas em paralelo (Fase 3 da Proposta D)
        toolResults := v.executor.ExecuteBatch(ctx, resp.ToolCalls)

        // 4. OBSERVE: Adiciona resultados ao histórico
        state.History = append(state.History, llm.Message{Role: "assistant", ToolCalls: resp.ToolCalls})
        for _, res := range toolResults {
            state.History = append(state.History, llm.Message{
                Role: "tool",
                ToolCallID: res.ID,
                Content: res.Output,
            })
        }

        state.Steps++
    }
    return fmt.Errorf("agent reached max steps without resolution")
}
```

---

## 3. Integração com o Harness (Quality Gate)

O loop de raciocínio é o principal sujeito de teste do **Harness**. Para que o Harness valide a "inteligência", o `Agent` deve expor um **Trace de Raciocínio**.

### O que o Harness monitora no Loop:

- **Coerência do Plano**: O plano gerado no Step 1 faz sentido para a tarefa?
- **Alucinação de Ferramentas**: A LLM tentou inventar uma ferramenta que não existe no `internal/tools`?
- **Loop Infinito**: O agente ficou preso tentando `read_file` no mesmo arquivo sem progredir?
- **Recuperação de Erro**: Se a ferramenta `edit` falhar por erro de sintaxe, o agente consegue ler o erro e corrigir o código no próximo passo?

---

## 4. O Planner (Estratégia de Longo Prazo)

Para tarefas complexas (ex: "Refatore todo o módulo de storage"), o Vectora utiliza um sub-loop de **Planejamento**.

1. **Macro-Planejador**: Decompõe a tarefa em 5-10 sub-tarefas menores.
2. **Micro-Executor**: Executa o loop ReAct descrito acima para cada sub-tarefa.
3. **Verificador**: Após cada sub-tarefa, o Vectora roda um mini-Harness interno para garantir que a alteração não quebrou o build.

---

## 5. Considerações de Segurança (Sandbox)

Como o loop de raciocínio permite que a IA "atue" no sistema:

- **Trust Folder**: O agente só pode ler/escrever dentro do diretório do projeto.
- **Approval Gate**: Ferramentas sensíveis (como `rm` ou `git push`) exigem confirmação manual no terminal ou VSCode, a menos que o modo `--force` esteja ativo.
- **Token Limit**: O loop é interrompido se o custo da tarefa exceder o limite definido no `config.yaml`.
