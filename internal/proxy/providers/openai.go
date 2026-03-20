package providers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/llmgw/internal/domain"
)

// OpenAIProvider handles OpenAI-compatible APIs (OpenAI, DeepSeek, Alibaba Qwen, etc.)
type OpenAIProvider struct {
	apiKey  string
	baseURL string
}

func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	return &OpenAIProvider{apiKey: apiKey, baseURL: baseURL}
}

func (p *OpenAIProvider) Complete(ctx context.Context, userID string, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	// TODO: POST {baseURL}/chat/completions with OpenAI request format
	// Return unified ChatResponse
	return nil, nil
}

func (p *OpenAIProvider) Stream(c *gin.Context, userID string, req *domain.ChatRequest, q QuotaDeductor, logger ChatLogger) {
	// TODO: stream SSE from provider → forward to client, then deduct & log
}