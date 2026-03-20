package providers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/llmgw/internal/domain"
)

// AnthropicProvider handles the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey string
}

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{apiKey: apiKey}
}

func (p *AnthropicProvider) Complete(ctx context.Context, userID string, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	// TODO: convert OpenAI-format messages → Anthropic format, call API, convert response back
	return nil, nil
}

func (p *AnthropicProvider) Stream(c *gin.Context, userID string, req *domain.ChatRequest, q QuotaDeductor, logger ChatLogger) {
	// TODO: Anthropic streaming (server-sent events)
}