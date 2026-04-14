package llm

import (
	"context"
	"fmt"
)

type Router struct {
	Providers        map[string]Provider
	DefaultProvider  string
	FallbackProvider string
	FallbackModels   map[string]string
}

func NewRouter() *Router {
	return &Router{
		Providers:      make(map[string]Provider),
		FallbackModels: make(map[string]string),
	}
}

func (r *Router) RegisterProvider(name string, p Provider, asDefault bool) {
	r.Providers[name] = p
	if asDefault || r.DefaultProvider == "" {
		r.DefaultProvider = name
	}
}

func (r *Router) SetFallbackProvider(name string) {
	r.FallbackProvider = name
}

func (r *Router) SetFallbackModel(providerName, model string) {
	r.FallbackModels[providerName] = model
}

func (r *Router) GetFallbackModel(providerName string) string {
	return r.FallbackModels[providerName]
}

func (r *Router) ListModels(ctx context.Context, providerName string) ([]string, error) {
	if providerName == "" {
		p := r.GetDefault()
		if p == nil {
			return nil, fmt.Errorf("no default provider configured")
		}
		return p.ListModels(ctx)
	}

	p, err := r.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	return p.ListModels(ctx)
}

func (r *Router) GetProvider(name string) (Provider, error) {
	if p, ok := r.Providers[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("provider %s not found", name)
}

func (r *Router) GetDefault() Provider {
	if r.DefaultProvider == "" {
		return nil
	}
	return r.Providers[r.DefaultProvider]
}

func (r *Router) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	p := r.GetDefault()
	if p == nil {
		return CompletionResponse{}, fmt.Errorf("no LLM provider configured")
	}

	res, err := p.Complete(ctx, req)
	if err != nil && r.FallbackProvider != "" && r.FallbackProvider != r.DefaultProvider {
		// Try fallback provider (e.g., Gemini)
		fp, ok := r.Providers[r.FallbackProvider]
		if ok && fp.IsConfigured() {
			// Optional: log that we are falling back
			return fp.Complete(ctx, req)
		}
	}

	return res, err
}

func (r *Router) Embed(ctx context.Context, input string, model string) ([]float32, error) {
	family := FamilyFromModel(model)

	// nativeEmbeddingProviders: families that MUST use their own API for embeddings.
	// These never fall through to a gateway (openrouter/anannas can't serve them).
	nativeEmbeddingProviders := map[string]string{
		"google": "gemini",
		"openai": "openai",
		"qwen":   "qwen",
		"voyage": "voyage",
	}

	// 1. Native provider — if the model belongs to a native family, use it exclusively.
	//    Do NOT fall through to gateway on failure (gateways don't support embedding APIs).
	if providerName, ok := nativeEmbeddingProviders[family]; ok {
		if p, err := r.GetProvider(providerName); err == nil && p.IsConfigured() {
			return p.Embed(ctx, input, model)
		}
		// Native provider not configured — skip gateway, go straight to fallbacks.
	}

	// 2. Default provider (only if it is NOT a gateway provider).
	if p := r.GetDefault(); p != nil {
		name := p.Name()
		if name != "openrouter" && name != "anannas" {
			if vec, err := p.Embed(ctx, input, model); err == nil {
				return vec, nil
			}
		}
	}

	// 3. Fallback chain: gemini → openai → voyage.
	for _, fb := range []string{"gemini", "openai", "voyage"} {
		if p, err := r.GetProvider(fb); err == nil && p.IsConfigured() {
			if vec, err := p.Embed(ctx, input, model); err == nil {
				return vec, nil
			}
		}
	}

	return nil, fmt.Errorf("no embedding provider available for model %q (tried native + fallbacks)", model)
}

func (r *Router) IsConfigured() bool {
	return r.GetDefault() != nil && r.GetDefault().IsConfigured()
}
