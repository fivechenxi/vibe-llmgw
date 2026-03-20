package proxy

import (
	"fmt"

	"github.com/yourorg/llmgw/internal/config"
	"github.com/yourorg/llmgw/internal/proxy/providers"
)

// Provider is the unified interface every LLM backend must implement.
type Provider interface {
	providers.Provider
}

type Router struct {
	routes map[string]Provider
}

func NewRouter(cfg *config.Config) *Router {
	r := &Router{routes: make(map[string]Provider)}

	openai := providers.NewOpenAIProvider(cfg.Providers.OpenAI.APIKey, cfg.Providers.OpenAI.BaseURL)
	for _, m := range []string{"gpt-4o", "gpt-4o-mini"} {
		r.routes[m] = openai
	}

	anthropic := providers.NewAnthropicProvider(cfg.Providers.Anthropic.APIKey, cfg.Proxy)
	for _, m := range []string{"claude-3-5-sonnet", "claude-3-haiku"} {
		r.routes[m] = anthropic
	}

	deepseek := providers.NewOpenAIProvider(cfg.Providers.DeepSeek.APIKey, cfg.Providers.DeepSeek.BaseURL)
	for _, m := range []string{"deepseek-v3", "deepseek-r1"} {
		r.routes[m] = deepseek
	}

	alibaba := providers.NewOpenAIProvider(cfg.Providers.Alibaba.APIKey, cfg.Providers.Alibaba.BaseURL)
	for _, m := range []string{"qwen-max", "qwen-plus"} {
		r.routes[m] = alibaba
	}

	return r
}

func (r *Router) Get(modelID string) (Provider, error) {
	p, ok := r.routes[modelID]
	if !ok {
		return nil, fmt.Errorf("no provider for model %q", modelID)
	}
	return p, nil
}