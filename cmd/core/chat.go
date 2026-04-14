package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Kaffyn/Vectora/internal/api/ipc"
	"github.com/Kaffyn/Vectora/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start a rich TUI chat with Vectora",
	Long:  "Launch a Bubbletea-based REPL session that maintains conversation history natively within the core.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ensureCoreConnected()
		if err != nil {
			return fmt.Errorf("core not connected: %v\nPlease try running: vectora start", err)
		}
		defer client.Close()

		cwd, _ := os.Getwd()
		absCwd, _ := filepath.Abs(cwd)
		conversationID := workspaceConversationID(absCwd)

		// Define the responder logic for the TUI
		responder := func(input string) (string, error) {
			ctx := context.Background()
			appendConversationEntry(conversationID, "user", input)

			req := map[string]string{
				"workspace_id":    conversationID,
				"query":           input,
				"conversation_id": conversationID,
			}

			var resp struct {
				Answer string `json:"answer"`
			}

			err := client.Send(ctx, "workspace.query", req, &resp)
			if err != nil {
				return "", err
			}

			appendConversationEntry(conversationID, "assistant", resp.Answer)
			return resp.Answer, nil
		}

		m := tui.InitialChatModel(responder)
		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("failed to run TUI: %v", err)
		}

		return nil
	},
}
