package proxy

import (
	"testing"

	"github.com/yourorg/llmgw/internal/config"
	"github.com/yourorg/llmgw/internal/proxy/providers"
)

func newTestRouter() *Router {
	cfg := &config.Config{} // empty config: no real API keys needed
	return NewRouter(cfg)
}

func TestRouterGet_KnownModels(t *testing.T) {
	r := newTestRouter()

	known := []string{
		"mock",
		"gpt-4o", "gpt-4o-mini",
		"claude-3-5-sonnet", "claude-3-haiku", "claude-haiku-4-5",
		"deepseek-v3", "deepseek-r1",
		"qwen-max", "qwen-plus",
	}
	for _, m := range known {
		p, err := r.Get(m)
		if err != nil {
			t.Errorf("Get(%q) unexpected error: %v", m, err)
		}
		if p == nil {
			t.Errorf("Get(%q) returned nil provider", m)
		}
	}
}

func TestRouterGet_UnknownModel(t *testing.T) {
	r := newTestRouter()
	_, err := r.Get("not-a-model")
	if err == nil {
		t.Error("expected error for unknown model, got nil")
	}
}

func TestRouterRegister(t *testing.T) {
	r := newTestRouter()
	mock := providers.NewMockProvider()
	r.Register("custom-model", mock)

	p, err := r.Get("custom-model")
	if err != nil {
		t.Fatalf("Get after Register returned error: %v", err)
	}
	if p != mock {
		t.Error("expected registered provider to be returned")
	}
}

func TestRouterRegister_Override(t *testing.T) {
	r := newTestRouter()
	custom := providers.NewMockProvider()
	custom.Response = "overridden"
	r.Register("gpt-4o", custom)

	p, err := r.Get("gpt-4o")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != custom {
		t.Error("expected override provider to be returned")
	}
}
