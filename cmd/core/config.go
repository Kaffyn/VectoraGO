package main

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Kaffyn/Vectora/internal/storage/infra"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Vectora core configurations",
	Long:  "Read or modify global configuration keys stored in official OS locations (config.yaml)",
	Run: func(cmd *cobra.Command, args []string) {
		runConfigInteractive()
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [KEY]",
	Short: "Get a configuration key value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := strings.ToLower(args[0])
		cfg := infra.LoadConfig()

		var val string
		switch key {
		case "gemini_api_key": val = cfg.Providers.Gemini.APIKey
		case "claude_api_key": val = cfg.Providers.Claude.APIKey
		case "openai_api_key": val = cfg.Providers.OpenAI.APIKey
		case "default_provider": val = cfg.Settings.DefaultProvider
		case "active_gateway": val = cfg.Settings.ActiveGateway
		default:
			fmt.Printf("Key '%s' not recognized.\n", key)
			return
		}

		if val == "" {
			fmt.Println("Value not set.")
		} else {
			fmt.Printf("%s=%s\n", key, val)
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [KEY] [VALUE]",
	Short: "Set a configuration key",
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			runConfigInteractive()
			return
		}

		key := strings.ToLower(args[0])
		var value string
		if len(args) == 2 {
			value = args[1]
		} else {
			prompt := &survey.Password{Message: fmt.Sprintf("Enter value for %s:", key)}
			survey.AskOne(prompt, &value)
		}

		cfg := infra.LoadConfig()
		switch key {
		case "gemini_api_key": cfg.Providers.Gemini.APIKey = value
		case "claude_api_key": cfg.Providers.Claude.APIKey = value
		case "openai_api_key": cfg.Providers.OpenAI.APIKey = value
		case "default_provider": cfg.Settings.DefaultProvider = value
		case "active_gateway": cfg.Settings.ActiveGateway = value
		default:
			fmt.Printf("Error: Key '%s' not recognized.\n", key)
			return
		}

		if err := infra.SaveConfig(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
		} else {
			fmt.Println("Config updated successfully.")
		}
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigInteractive() {
	cfg := infra.LoadConfig()

	providers := []string{"gemini", "claude", "openai", "qwen", "mistral", "deepseek"}
	
	var provider string
	if err := survey.AskOne(&survey.Select{
		Message: "Select default AI provider:",
		Options: providers,
		Default: cfg.Settings.DefaultProvider,
	}, &provider); err == nil {
		cfg.Settings.DefaultProvider = provider
	}

	var apiKey string
	msg := fmt.Sprintf("Enter API Key for %s (leave empty to keep current):", provider)
	if err := survey.AskOne(&survey.Password{Message: msg}, &apiKey); err == nil && apiKey != "" {
		switch provider {
		case "gemini": cfg.Providers.Gemini.APIKey = apiKey
		case "claude": cfg.Providers.Claude.APIKey = apiKey
		case "openai": cfg.Providers.OpenAI.APIKey = apiKey
		}
	}

	if err := infra.SaveConfig(cfg); err != nil {
		fmt.Printf("Error saving: %v\n", err)
	} else {
		fmt.Println("Configuration saved to config.yaml")
	}
}
