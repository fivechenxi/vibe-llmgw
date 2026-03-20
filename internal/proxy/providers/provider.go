package providers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/llmgw/internal/domain"
)

// Provider is the unified interface all LLM backends implement.
// Both streaming and non-streaming paths are required.
type Provider interface {
	Complete(ctx context.Context, userID string, req *domain.ChatRequest) (*domain.ChatResponse, error)
	Stream(c *gin.Context, userID string, req *domain.ChatRequest, quotaDeductor QuotaDeductor, logger ChatLogger)
}

// QuotaDeductor is a narrow interface so providers can deduct tokens without importing the quota package.
type QuotaDeductor interface {
	Deduct(ctx context.Context, userID, modelID string, tokens int) error
}

// ChatLogger is a narrow interface so providers can persist logs without importing the chat package.
type ChatLogger interface {
	Save(ctx context.Context, log *domain.ChatLog) error
}