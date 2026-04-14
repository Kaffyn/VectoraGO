package infra

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds the runtime configuration for Vectora.
type Config struct {
	Providers struct {
		Gemini   ProviderConfig `yaml:"gemini"`
		Claude   ProviderConfig `yaml:"claude"`
		OpenAI   ProviderConfig `yaml:"openai"`
		Qwen     ProviderConfig `yaml:"qwen"`
		OpenRouter ProviderConfig `yaml:"openrouter"`
		Anannas  ProviderConfig `yaml:"anannas"`
		DeepSeek ProviderConfig `yaml:"deepseek"`
		Mistral  ProviderConfig `yaml:"mistral"`
		Grok     ProviderConfig `yaml:"grok"`
		Zhipu    ProviderConfig `yaml:"zhipu"`
		Meta     ProviderConfig `yaml:"meta"`
		Voyage   ProviderConfig `yaml:"voyage"`
	} `yaml:"providers"`

	Settings struct {
		DefaultProvider   string `yaml:"default_provider"`
		DefaultModel      string `yaml:"default_model"`
		EmbeddingModel    string `yaml:"embedding_model"`
		FallbackProvider  string `yaml:"fallback_provider"`
		ActiveGateway     string `yaml:"active_gateway"`
		LogLevel          string `yaml:"log_level"`
		SecretKey         string `yaml:"secret_key"`
	} `yaml:"settings"`
}

type ProviderConfig struct {
	APIKey        string `yaml:"api_key"`
	BaseURL       string `yaml:"base_url"`
	FallbackModel string `yaml:"fallback_model"`
}

// GetConfigPath returns the path to the config.yaml file.
func GetConfigPath() string {
	appData := os.Getenv("APPDATA")
	userProfile, _ := os.UserHomeDir()

	var root string
	if appData != "" {
		root = filepath.Join(appData, "Vectora")
	} else if userProfile != "" {
		root = filepath.Join(userProfile, ".Vectora")
	} else {
		root = "."
	}
	return filepath.Join(root, "config.yaml")
}

// LoadConfig loads configuration from yaml, with .env as legacy fallback.
func LoadConfig() *Config {
	path := GetConfigPath()
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}

	cfg := &Config{}

	// 1. Try Loading YAML
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, cfg); err == nil {
			return cfg
		}
	}

	// 2. Legacy .env Fallback
	envPath := filepath.Join(dir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		godotenv.Load(envPath)
		cfg.Providers.Gemini.APIKey = os.Getenv("GEMINI_API_KEY")
		cfg.Providers.Claude.APIKey = os.Getenv("CLAUDE_API_KEY")
		cfg.Providers.OpenAI.APIKey = os.Getenv("OPENAI_API_KEY")
		cfg.Providers.Voyage.APIKey = os.Getenv("VOYAGE_API_KEY")
		cfg.Settings.DefaultProvider = os.Getenv("DEFAULT_PROVIDER")
		cfg.Settings.ActiveGateway = os.Getenv("ACTIVE_GATEWAY")
		
		// Auto-migrate to YAML on next save
	}

	return cfg
}

// SaveConfig persists configuration to config.yaml.
func SaveConfig(cfg *Config) error {
	path := GetConfigPath()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
