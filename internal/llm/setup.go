package llm

import (
	"context"
	"github.com/Kaffyn/Vectora/internal/storage/infra"
)

// SetupRouter initializes a router with all available providers based on configuration.
func SetupRouter(ctx context.Context, cfg *infra.Config, prefs *infra.UserPreferences) *Router {
	router := NewRouter()
	router.SetFallbackProvider("gemini")
	if cfg.Settings.FallbackProvider != "" {
		router.SetFallbackProvider(cfg.Settings.FallbackProvider)
	}

	// Default fallback models (April 2026 Standards)
	router.SetFallbackModel("gemini", "gemini-3-flash-preview")
	router.SetFallbackModel("claude", "claude-4.6-sonnet")
	router.SetFallbackModel("openai", "gpt-5.4-mini")
	router.SetFallbackModel("qwen", "qwen3.6-turbo")

	// Set fallback models from config (overrides defaults)
	if cfg.Providers.Gemini.FallbackModel != "" {
		router.SetFallbackModel("gemini", cfg.Providers.Gemini.FallbackModel)
	}
	if cfg.Providers.Claude.FallbackModel != "" {
		router.SetFallbackModel("claude", cfg.Providers.Claude.FallbackModel)
	}
	if cfg.Providers.OpenAI.FallbackModel != "" {
		router.SetFallbackModel("openai", cfg.Providers.OpenAI.FallbackModel)
	}
	if cfg.Providers.Qwen.FallbackModel != "" {
		router.SetFallbackModel("qwen", cfg.Providers.Qwen.FallbackModel)
	}

	// Use preference for default provider if available
	defaultProv := prefs.DefaultProvider

	// Se há um gateway ativo, ele se torna o provider padrão para completion
	// Os providers nativos ainda são registrados para embeddings
	if cfg.Settings.ActiveGateway == "openrouter" && cfg.Providers.OpenRouter.APIKey != "" {
		defaultProv = "openrouter"
	} else if cfg.Settings.ActiveGateway == "anannas" && cfg.Providers.Anannas.APIKey != "" {
		defaultProv = "anannas"
	}

	// Register Native Providers
	if cfg.Providers.Gemini.APIKey != "" {
		p, _ := NewGeminiProvider(ctx, cfg.Providers.Gemini.APIKey)
		router.RegisterProvider("gemini", p, defaultProv == "gemini")
	}
	if cfg.Providers.Claude.APIKey != "" {
		p, _ := NewClaudeProvider(ctx, cfg.Providers.Claude.APIKey)
		router.RegisterProvider("claude", p, defaultProv == "claude")
	}
	if cfg.Providers.OpenAI.APIKey != "" {
		p := NewOpenAIProvider(cfg.Providers.OpenAI.APIKey, cfg.Providers.OpenAI.BaseURL, "openai")
		router.RegisterProvider("openai", p, defaultProv == "openai")
	}
	if cfg.Providers.Qwen.APIKey != "" {
		p := NewOpenAIProvider(cfg.Providers.Qwen.APIKey, cfg.Providers.Qwen.BaseURL, "qwen")
		router.RegisterProvider("qwen", p, defaultProv == "qwen")
	}

	// Register Embedding-Specialized Providers
	if cfg.Providers.Voyage.APIKey != "" {
		p, err := NewVoyageProvider(ctx, cfg.Providers.Voyage.APIKey)
		if err == nil {
			router.RegisterProvider("voyage", p, defaultProv == "voyage")
		}
	}

	// Register Gateway Providers
	if cfg.Providers.OpenRouter.APIKey != "" {
		p := NewGatewayProvider(cfg.Providers.OpenRouter.APIKey, "https://openrouter.ai/api/v1", "openrouter")
		router.RegisterProvider("openrouter", p, defaultProv == "openrouter")
	}
	if cfg.Providers.Anannas.APIKey != "" {
		p := NewGatewayProvider(cfg.Providers.Anannas.APIKey, "https://api.anannas.ai/v1", "anannas")
		router.RegisterProvider("anannas", p, defaultProv == "anannas")
	}

	return router
}
