# 🌈 Charm Ecosystem Analysis (April 2026)

Este documento analisa o ecossistema da [Charm.sh](https://charm.sh)

## 🏛️ Alinhamento Arquitetura (Proposta D)

Na proposta de arquitetura do Vectora, o ecossistema Charm atua como a interface de alta fidelidade entre o **Core Daemon** e o usuário, garantindo que a simplicidade do Go no backend seja traduzida em elegância no frontend.

---

## 🏗️ Simbiose Cobra-Charm (Estratégia Tática)

Para garantir performance industrial com UX premium, seguiremos três pilares:

### 1. The Log Splitter (Dev vs. Ops)

O sistema de logs será um wrapper sobre `charmbracelet/log`:

- **Modo TTY (Terminal)**: Estilos ricos, cores e indentação para o desenvolvedor local.
- **Modo Pipeline (Kubernetes/File)**: Estilos desativados e foco em **JSON estruturado** para o arquivo `vectora-core.log`.

### 2. Ponte Estática (Lip Gloss + Cobra)

Não limitaremos a beleza à tela cheia (Alternate Screen). Comandos Cobra padrão usarão **Lip Gloss** para tabelas e **Bubbles** (como Progress Bars) diretamente no stdout, elevando a qualidade de comandos como `vectora harness run`.

### 3. Concorrência via `tea.Cmd`

Como o Core é intensivo em I/O (RAG/Embeddings), utilizaremos `tea.Cmd` para disparar processos pesados. Isso mantém a UI fluida e responsiva, evitando "freezes" durante o processamento de grandes codebases.

---

## 🎨 Identidade Visual (Design System)

### 1. [Bubble Tea](https://github.com/charmbracelet/bubbletea) (The Engine)

- **Papel**: O framework fundamental baseado na arquitetura Elm. Ele gerencia o estado da TUI, as atualizações e a renderização.
- **Aplicação no Vectora**: Será o orquestrador de todas as telas interativas (Chat, Monitor de Indexação, Logs em tempo real).
- **Vantagem**: Concorrência nativa (comandos via Goroutines), facilitando a integração com o Core do Vectora que já opera em background.

### 2. [Lip Gloss](https://github.com/charmbracelet/lipgloss) (The Stylist)

- **Papel**: Definições de estilo estilo CSS para o terminal. Gerencia cores, bordas, padding, margens e layouts (layouts flexíveis).
- **Aplicação no Vectora**: Padronização da identidade visual:
  - **Primary**: Indigo/Purple (Vectora Brand)
  - **Success**: Spring Green
  - **Error**: Soft Red
- **Vantagem**: Permite criar componentes reutilizáveis e layouts adaptáveis a diferentes tamanhos de terminal.

### 3. [Bubbles](https://github.com/charmbracelet/bubbles) (The UI Kit)

- **Papel**: Componentes prontos para uso (inputs, textareas, viewports, spinners, progress bars).
- **Aplicação no Vectora**:
  - `Viewport`: Para scroll infinito no chat.
  - `Progress`: Para feedback visual de indexação RAG.
  - `Spinner`: Para estados de "Pensando..." do agente.

### 4. [Huh?](https://github.com/charmbracelet/huh) (The Form Builder)

- **Papel**: Um DSL de alto nível para criar formulários interativos e acessíveis.
- **Aplicação no Vectora**: Centralizar a configuração:
  - Configuração de chaves de API (Gemini, Claude, OpenAI).
  - Seleção de modelos e parâmetros de temperatura.
  - Configuração de caminhos de workspace.
- **Vantagem**: Transforma o `vectora config` de uma experiência frustrante de arquivos `.env` em um wizard premium.

### 5. [Glamour](https://github.com/charmbracelet/glamour) (The Markdown Renderer)

- **Papel**: Renderizador de Markdown com suporte a syntax highlighting para blocos de código.
- **Aplicação no Vectora**: Renderizar as respostas do agente. Como o Vectora é focado em código, o Glamour é essencial para que os diffs e snippets sugeridos sejam legíveis.

### 6. [Log](https://github.com/charmbracelet/log) (The Observer)

- **Papel**: Logging estruturado, bonito e pronto para produção.
- **Aplicação no Vectora**:
  - **Dev**: Logs coloridos e legíveis com timestamps e níveis.
  - **Prod (Kubernetes)**: Saída JSON estruturada para integração com Fluentd/Promtail/Loki.
- **Integração**: Substituirá as chamadas diretas de `fmt.Println` no Core por logs estruturados com campos como `trace_id`, `tenant_id` e `latency`.

---

## 🚀 Proposta de Arquitetura TUI para o Vectora

O objetivo é criar uma **Dashboard Unificada** que substitua a CLI estática atual:

| Área          | Tecnologia            | Descrição                                          |
| :------------ | :-------------------- | :------------------------------------------------- |
| **Sidebar**   | Lip Gloss             | Status do Core, Uso de Tokens, Provedor Ativo.     |
| **Main View** | Bubble Tea + Viewport | Chat REPL com histórico persistente.               |
| **Input**     | Huh? / Text Area      | Entrada de comandos com autocompletar de arquivos. |
| **Overlays**  | Huh?                  | Modais de configuração e help (F1).                |

## 🛠️ Próximos Passos (Sprint de UI)

1.  **Refatorar `internal/logger`**: Mudar para `charmbracelet/log` com suporte a `ENV_MODE=production`.
2.  **Criar `internal/tui/app.go`**: Definir o modelo raiz do Bubble Tea.
3.  **Implementar `vectora config` via Huh**: Substituir o survey/manual atual.
4.  **Polimento Visual**: Usar Lip Gloss para criar uma moldura (frame) premium ao redor do chat.

---

_Documento criado em Abril de 2026 para a especificação do Vectora 1.2 "Charmland"_
