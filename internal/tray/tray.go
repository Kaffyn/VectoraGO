//go:build windows

package tray

import (
	"context"

	"github.com/Kaffyn/Vectora/assets"
	"github.com/Kaffyn/Vectora/internal/config/i18n"
	"github.com/Kaffyn/Vectora/internal/storage/infra"
	"github.com/Kaffyn/Vectora/internal/llm"
	"github.com/getlantern/systray"
)

// ProviderInfo describes an LLM provider available in the tray menu
type ProviderInfo struct {
	ID      string                                                             // Internal identifier (gemini, claude, openai, etc.)
	Label   string                                                             // Display label (from i18n key)
	I18nKey string                                                             // i18n translation key
	GetKey  func(*infra.Config) string                                         // Function to get API key from config
	Setup   func(context.Context, string, *infra.Config) (llm.Provider, error) // Constructor
}

// AllProviders defines all 10+ LLM families supported by Vectora (AGENTS.md April 2026)
var AllProviders = []ProviderInfo{
	{
		ID:      "gemini",
		I18nKey: "tray_prov_gemini",
		GetKey: func(cfg *infra.Config) string {
			return cfg.Providers.Gemini.APIKey
		},
		Setup: func(ctx context.Context, key string, cfg *infra.Config) (llm.Provider, error) {
			return llm.NewGeminiProvider(ctx, key)
		},
	},
	{
		ID:      "claude",
		I18nKey: "tray_prov_claude",
		GetKey: func(cfg *infra.Config) string {
			return cfg.Providers.Claude.APIKey
		},
		Setup: func(ctx context.Context, key string, cfg *infra.Config) (llm.Provider, error) {
			return llm.NewClaudeProvider(ctx, key)
		},
	},
	{
		ID:      "openai",
		I18nKey: "tray_prov_openai",
		GetKey: func(cfg *infra.Config) string {
			return cfg.Providers.OpenAI.APIKey
		},
		Setup: func(ctx context.Context, key string, cfg *infra.Config) (llm.Provider, error) {
			return llm.NewOpenAIProvider(key, cfg.Providers.OpenAI.BaseURL, "openai"), nil
		},
	},
	{
		ID:      "deepseek",
		I18nKey: "tray_prov_deepseek",
		GetKey: func(cfg *infra.Config) string {
			return cfg.Providers.DeepSeek.APIKey
		},
		Setup: func(ctx context.Context, key string, cfg *infra.Config) (llm.Provider, error) {
			baseURL := cfg.Providers.DeepSeek.BaseURL
			if baseURL == "" {
				baseURL = "https://api.deepseek.com/v1"
			}
			return llm.NewOpenAIProvider(key, baseURL, "deepseek"), nil
		},
	},
	{
		ID:      "mistral",
		I18nKey: "tray_prov_mistral",
		GetKey: func(cfg *infra.Config) string {
			return cfg.Providers.Mistral.APIKey
		},
		Setup: func(ctx context.Context, key string, cfg *infra.Config) (llm.Provider, error) {
			baseURL := cfg.Providers.Mistral.BaseURL
			if baseURL == "" {
				baseURL = "https://api.mistral.ai/v1"
			}
			return llm.NewOpenAIProvider(key, baseURL, "mistral"), nil
		},
	},
	{
		ID:      "grok",
		I18nKey: "tray_prov_grok",
		GetKey: func(cfg *infra.Config) string {
			return cfg.Providers.Grok.APIKey
		},
		Setup: func(ctx context.Context, key string, cfg *infra.Config) (llm.Provider, error) {
			baseURL := cfg.Providers.Grok.BaseURL
			if baseURL == "" {
				baseURL = "https://api.x.ai/v1"
			}
			return llm.NewOpenAIProvider(key, baseURL, "grok"), nil
		},
	},
	{
		ID:      "zhipu",
		I18nKey: "tray_prov_zhipu",
		GetKey: func(cfg *infra.Config) string {
			return cfg.Providers.Zhipu.APIKey
		},
		Setup: func(ctx context.Context, key string, cfg *infra.Config) (llm.Provider, error) {
			baseURL := cfg.Providers.Zhipu.BaseURL
			if baseURL == "" {
				baseURL = "https://open.bigmodel.cn/api/paas/v4"
			}
			return llm.NewOpenAIProvider(key, baseURL, "zhipu"), nil
		},
	},
}

// GatewayInfo describes a routing gateway (openrouter, anannas, none)
type GatewayInfo struct {
	ID    string
	Label string
	GetKey func(*infra.Config) string
	Setup  func(context.Context, string, *infra.Config) (llm.Provider, error)
}

var AllGateways = []GatewayInfo{
	{
		ID:    "none",
		Label: "Nenhum (direto)",
		GetKey: func(cfg *infra.Config) string { return "" },
		Setup:  nil,
	},
	{
		ID:    "openrouter",
		Label: "OpenRouter",
		GetKey: func(cfg *infra.Config) string { return cfg.Providers.OpenRouter.APIKey },
		Setup: func(ctx context.Context, key string, cfg *infra.Config) (llm.Provider, error) {
			return llm.NewGatewayProvider(key, "https://openrouter.ai/api/v1", "openrouter"), nil
		},
	},
	{
		ID:    "anannas",
		Label: "Anannas",
		GetKey: func(cfg *infra.Config) string { return cfg.Providers.Anannas.APIKey },
		Setup: func(ctx context.Context, key string, cfg *infra.Config) (llm.Provider, error) {
			return llm.NewGatewayProvider(key, "https://api.anannas.ai/v1", "anannas"), nil
		},
	},
}

var (
	mStatus        *systray.MenuItem
	mGateway       *systray.MenuItem
	gatewayItems   map[string]*systray.MenuItem // Gateway menu items
	mProv          *systray.MenuItem
	providerItems  map[string]*systray.MenuItem // Dynamic provider menu items
	mModel         *systray.MenuItem
	modelItems     map[string]*systray.MenuItem // LLM model menu items
	mEmbedding     *systray.MenuItem
	embeddingItems map[string]*systray.MenuItem // Embedding model menu items
	mTurboQuant    *systray.MenuItem
	mLang          *systray.MenuItem
	mQuit          *systray.MenuItem

	ActiveGatewayID      string
	ActiveEmbeddingModel string

	mEn *systray.MenuItem
	mPt *systray.MenuItem
	mEs *systray.MenuItem
	mFr *systray.MenuItem

	ActiveProvider   llm.Provider
	ActiveProviderID string
	ActiveModel      string
)

// EmbeddingModels: always-available fallbacks + provider-specific extras.
// Gemini embedding 2 preview and Voyage 3 are always shown regardless of active LLM.
var alwaysEmbeddingModels = []string{
	"text-embedding-3-large",     // OpenAI fallback
	"gemini-embedding-2-preview", // Gemini Embedding 2 preview
	"voyage-3",                   // Voyage fallback
}

// providerEmbeddingModels lists extra embedding models unlocked by each provider.
// These appear IN ADDITION to alwaysEmbeddingModels when that provider is active.
var providerEmbeddingModels = map[string][]string{
	"gemini":  {"text-embedding-004"},
	"openai":  {"text-embedding-3-small"},
	"qwen":    {"qwen3-embedding-8b", "qwen3-embedding-4b", "qwen3-embedding-0.6b"},
	"zhipu":   {"embedding-3"},
	"mistral": {"mistral-embed"},
}

// activeEmbeddingModels returns the full list of embedding models for the current provider.
func activeEmbeddingModels() []string {
	seen := map[string]bool{}
	result := []string{}
	// Provider-specific first (more relevant)
	for _, m := range providerEmbeddingModels[ActiveProviderID] {
		if !seen[m] {
			result = append(result, m)
			seen[m] = true
		}
	}
	// Always-available fallbacks
	for _, m := range alwaysEmbeddingModels {
		if !seen[m] {
			result = append(result, m)
			seen[m] = true
		}
	}
	return result
}

// ProviderModels defines available models per provider (AGENTS.md April 2026)
var ProviderModels = map[string][]string{
	"gemini":     {"gemini-3.1-pro-preview", "gemini-3-flash-preview", "gemma-4-31b"},
	"claude":     {"claude-4.6-sonnet", "claude-4.6-opus", "claude-4.5-haiku"},
	"openai":     {"gpt-5.4-pro", "gpt-5.4-mini", "gpt-5-o1"},
	"openrouter": {"google/gemini-3.1-pro", "anthropic/claude-4.6-sonnet", "meta-llama/llama-4-70b"},
	"anannas":    {"anthropic/claude-4.6-sonnet", "google/gemini-3.1-pro", "openai/gpt-5.4-pro"},
	"deepseek":   {"deepseek-v3.2", "deepseek-v3.2-speciale"},
	"mistral":    {"mistral-large-3", "mistral-small-4"},
	"grok":       {"grok-4.20", "grok-4.1"},
	"zhipu":      {"glm-5.1", "glm-5-flash"},
}

// ReloadActiveProvider re-reads the .env configuration from disk and re-initializes
// the active LLM provider. This is critical for picking up API keys set by the CLI
// without requiring a full daemon restart.
func ReloadActiveProvider() {
	cfg := infra.LoadConfig()
	prefs := infra.LoadPreferences()

	// Load active model and provider from preferences
	ActiveModel = prefs.ActiveModel
	if ActiveModel == "" {
		ActiveModel = cfg.Providers.Gemini.FallbackModel // Default to Gemini fallback if nothing else
	}

	activeProvID := prefs.DefaultProvider
	if activeProvID == "" {
		activeProvID = ActiveProviderID
	}

	// Try to refresh the preferred provider
	if activeProvID != "" {
		for _, prov := range AllProviders {
			if prov.ID == activeProvID {
				key := prov.GetKey(cfg)
				if key != "" {
					ActiveProviderID = prov.ID
					setProvider(prov, key, cfg)
					updateLabels()
					return
				}
			}
		}
	}

	// Fallback logic
	for _, prov := range AllProviders {
		key := prov.GetKey(cfg)
		if key != "" {
			ActiveProviderID = prov.ID
			setProvider(prov, key, cfg)
			updateLabels()
			return
		}
	}
}

// Setup configures and launches the systray.
func Setup() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(assets.IconData)
	systray.SetTitle("Vectora")
	systray.SetTooltip("Vectora - AI Assistant")

	// Status (informational only)
	mStatus = systray.AddMenuItem("", "")
	mStatus.Disable()
	systray.AddSeparator()

	// Gateway selection (openrouter, anannas, none)
	mGateway = systray.AddMenuItem("Gateway", "")
	gatewayItems = make(map[string]*systray.MenuItem)
	cfg0 := infra.LoadConfig()
	ActiveGatewayID = cfg0.Settings.ActiveGateway
	if ActiveGatewayID == "" {
		ActiveGatewayID = "none"
	}
	for _, gw := range AllGateways {
		item := mGateway.AddSubMenuItemCheckbox(gw.Label, "", gw.ID == ActiveGatewayID)
		gatewayItems[gw.ID] = item
	}

	// AI Provider selection
	mProv = systray.AddMenuItem("", "")
	providerItems = make(map[string]*systray.MenuItem)

	for _, prov := range AllProviders {
		item := mProv.AddSubMenuItemCheckbox("", "", prov.ID == ActiveProviderID)
		providerItems[prov.ID] = item
	}

	// LLM model selection (Dynamic based on provider)
	mModel = systray.AddMenuItem("LLM", "")
	modelItems = make(map[string]*systray.MenuItem)
	for _, provModels := range llm.ProviderModels {
		for _, model := range provModels {
			if _, exists := modelItems[model]; !exists {
				item := mModel.AddSubMenuItemCheckbox(model, "", false)
				item.Hide()
				modelItems[model] = item
			}
		}
	}

	// Embedding model selection (always: gemini-embedding-2-preview + voyage-3; extra per provider)
	mEmbedding = systray.AddMenuItem("Embedding", "")
	embeddingItems = make(map[string]*systray.MenuItem)
	prefs0 := infra.LoadPreferences()
	ActiveEmbeddingModel = prefs0.ActiveEmbeddingModel
	if ActiveEmbeddingModel == "" {
		ActiveEmbeddingModel = "gemini-embedding-2-preview"
	}
	// Pre-create all possible embedding models (shown/hidden dynamically)
	allEmbeddingModels := append([]string{}, alwaysEmbeddingModels...)
	for _, models := range providerEmbeddingModels {
		for _, m := range models {
			allEmbeddingModels = append(allEmbeddingModels, m)
		}
	}
	seen := map[string]bool{}
	for _, m := range allEmbeddingModels {
		if seen[m] {
			continue
		}
		seen[m] = true
		item := mEmbedding.AddSubMenuItemCheckbox(m, "", m == ActiveEmbeddingModel)
		item.Hide()
		embeddingItems[m] = item
	}

	systray.AddSeparator()
	mTurboQuant = systray.AddMenuItem("TurboQuant (Beta)", "Enable 10x vector compression")
	if infra.LoadPreferences().EnableTurboQuantBeta {
		mTurboQuant.Check()
	}

	// Language selection
	mLang = systray.AddMenuItem("", "")
	mEn = mLang.AddSubMenuItemCheckbox("English", "", false)
	mPt = mLang.AddSubMenuItemCheckbox("Português", "", false)
	mEs = mLang.AddSubMenuItemCheckbox("Español", "", false)
	mFr = mLang.AddSubMenuItemCheckbox("Français", "", false)

	switch infra.LoadPreferences().Language {
	case "en":
		mEn.Check()
	case "pt":
		mPt.Check()
	case "es":
		mEs.Check()
	case "fr":
		mFr.Check()
	}

	systray.AddSeparator()
	mQuit = systray.AddMenuItem("", "")

	updateLabels()

	// Initial config check for default provider
	ReloadActiveProvider()

	// Event loop
	go func() {
		for {
			select {
			// Provider selection
			case <-mQuit.ClickedCh:
				systray.Quit()
			case <-mEn.ClickedCh:
				mEn.Check()
				mPt.Uncheck()
				mEs.Uncheck()
				mFr.Uncheck()
				i18n.SetLanguage("en")
				p := infra.LoadPreferences()
				p.Language = "en"
				infra.SavePreferences(p)
				updateLabels()
			case <-mPt.ClickedCh:
				mPt.Check()
				mEn.Uncheck()
				mEs.Uncheck()
				mFr.Uncheck()
				i18n.SetLanguage("pt")
				p := infra.LoadPreferences()
				p.Language = "pt"
				infra.SavePreferences(p)
				updateLabels()
			case <-mEs.ClickedCh:
				mEs.Check()
				mEn.Uncheck()
				mPt.Uncheck()
				mFr.Uncheck()
				i18n.SetLanguage("es")
				p := infra.LoadPreferences()
				p.Language = "es"
				infra.SavePreferences(p)
				updateLabels()
			case <-mFr.ClickedCh:
				mFr.Check()
				mEn.Uncheck()
				mPt.Uncheck()
				mEs.Uncheck()
				i18n.SetLanguage("fr")
				p := infra.LoadPreferences()
				p.Language = "fr"
				infra.SavePreferences(p)
				updateLabels()

			// Dynamic gateway selection
			default:
				cfg := infra.LoadConfig()
				for id, item := range gatewayItems {
					select {
					case <-item.ClickedCh:
						ActiveGatewayID = id
						// Persist gateway choice
						envVal := id
						if id == "none" {
							envVal = ""
						}
						// Save via env map (reuse godotenv path from infra)
						saveGateway(envVal, cfg)
						// If gateway selected, set active provider to gateway
						for _, gw := range AllGateways {
							if gw.ID == id && gw.Setup != nil {
								key := gw.GetKey(cfg)
								if key != "" {
									p, err := gw.Setup(context.Background(), key, cfg)
									if err == nil {
										ActiveProvider = p
										ActiveProviderID = id
									}
								}
							}
						}
						updateLabels()
					default:
					}
				}

				// Dynamic provider selection
				for id, item := range providerItems {
					select {
					case <-item.ClickedCh:
						var selectedProv ProviderInfo
						for _, p := range AllProviders {
							if p.ID == id {
								selectedProv = p
								break
							}
						}

						ActiveProviderID = id
						p := infra.LoadPreferences()
						p.DefaultProvider = id
						infra.SavePreferences(p)

						key := selectedProv.GetKey(cfg)
						setProvider(selectedProv, key, cfg)
						updateLabels()
					default:
					}
				}

				// Dynamic LLM model selection
				for model, item := range modelItems {
					select {
					case <-item.ClickedCh:
						ActiveModel = model
						p := infra.LoadPreferences()
						p.ActiveModel = model
						infra.SavePreferences(p)
						updateLabels()
						infra.NotifyOS("Vectora", "LLM: "+model)
					default:
					}
				}

				// Dynamic embedding model selection
				for model, item := range embeddingItems {
					select {
					case <-item.ClickedCh:
						ActiveEmbeddingModel = model
						p := infra.LoadPreferences()
						p.ActiveEmbeddingModel = model
						infra.SavePreferences(p)
						updateLabels()
						infra.NotifyOS("Vectora", "Embedding: "+model)
					default:
					}
				}

				// TurboQuant selection
				select {
				case <-mTurboQuant.ClickedCh:
					p := infra.LoadPreferences()
					p.EnableTurboQuantBeta = !p.EnableTurboQuantBeta
					infra.SavePreferences(p)
					if p.EnableTurboQuantBeta {
						mTurboQuant.Check()
						infra.NotifyOS("Vectora", "TurboQuant (Beta) Habilitado")
					} else {
						mTurboQuant.Uncheck()
						infra.NotifyOS("Vectora", "TurboQuant (Beta) Desabilitado")
					}
					// Note: Real world effect requires new embedding session
					infra.NotifyOS("Info", "Note: Changes apply only to new embeddings.")
				default:
				}
			}
		}
	}()
}

func setProvider(prov ProviderInfo, secret string, cfg *infra.Config) {
	if secret == "" {
		infra.NotifyOS("Vectora", "API key for "+prov.ID+" not configured.")
		return
	}

	ctx := context.Background()
	p, err := prov.Setup(ctx, secret, cfg)
	if err != nil {
		infra.NotifyOS("Vectora", "Failed to initialize "+prov.ID+" provider: "+err.Error())
		return
	}

	ActiveProvider = p
}

func saveGateway(value string, cfg *infra.Config) {
	cfg.Settings.ActiveGateway = value
	_ = infra.SaveConfig(cfg)
}

func updateLabels() {
	mStatus.SetTitle(i18n.T("tray_status"))
	mGateway.SetTitle("Gateway")
	mProv.SetTitle(i18n.T("tray_provider"))
	mModel.SetTitle("LLM")
	mEmbedding.SetTitle("Embedding")

	for id, item := range gatewayItems {
		if id == ActiveGatewayID {
			item.Check()
		} else {
			item.Uncheck()
		}
	}

	for id, item := range providerItems {
		if id == ActiveProviderID {
			item.Check()
			item.SetTitle(id)
		} else {
			item.Uncheck()
			item.SetTitle(id)
		}
	}

	// Update LLM model visibility
	validModels := llm.ProviderModels[ActiveProviderID]
	validSet := map[string]bool{}
	for _, vm := range validModels {
		validSet[vm] = true
	}
	for m, item := range modelItems {
		if validSet[m] {
			item.Show()
			if m == ActiveModel {
				item.Check()
			} else {
				item.Uncheck()
			}
		} else {
			item.Hide()
		}
	}

	// Update embedding model visibility: always-available + provider-specific
	currentEmbedding := activeEmbeddingModels()
	embSet := map[string]bool{}
	for _, m := range currentEmbedding {
		embSet[m] = true
	}
	for m, item := range embeddingItems {
		if embSet[m] {
			item.Show()
			if m == ActiveEmbeddingModel {
				item.Check()
			} else {
				item.Uncheck()
			}
		} else {
			item.Hide()
		}
	}

	mLang.SetTitle(i18n.T("tray_language"))
	mQuit.SetTitle(i18n.T("tray_quit"))
}

func onExit() {
	if infra.Logger() != nil {
		infra.Logger().Info("Core shutting down...")
	}
}
