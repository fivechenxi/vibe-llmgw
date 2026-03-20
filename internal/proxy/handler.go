package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourorg/llmgw/internal/config"
	"github.com/yourorg/llmgw/internal/domain"
	"github.com/yourorg/llmgw/internal/middleware"
	"github.com/yourorg/llmgw/internal/quota"
)

// QuotaService is the interface the handler needs from the quota layer.
type QuotaService interface {
	Check(ctx context.Context, userID, modelID string) error
	Deduct(ctx context.Context, userID, modelID string, tokens int) error
}

// ChatSaver is the interface the handler needs to persist logs.
type ChatSaver interface {
	Save(ctx context.Context, log *domain.ChatLog) error
}

// ProviderRouter resolves a model ID to its Provider.
type ProviderRouter interface {
	Get(modelID string) (Provider, error)
}

type Handler struct {
	quotaSvc QuotaService
	chatSave ChatSaver
	router   ProviderRouter
}

func NewHandler(cfg *config.Config, quotaSvc QuotaService, chatSave ChatSaver) *Handler {
	return &Handler{
		quotaSvc: quotaSvc,
		chatSave: chatSave,
		router:   NewRouter(cfg),
	}
}

// newHandlerWithRouter is used in tests to inject a custom router.
func newHandlerWithRouter(quotaSvc QuotaService, chatSave ChatSaver, router ProviderRouter) *Handler {
	return &Handler{quotaSvc: quotaSvc, chatSave: chatSave, router: router}
}

func (h *Handler) Chat(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)

	var req domain.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Check quota
	if err := h.quotaSvc.Check(c.Request.Context(), userID, req.Model); err != nil {
		if errors.Is(err, quota.ErrQuotaExceeded) {
			c.JSON(http.StatusForbidden, gin.H{"error": "quota exceeded"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2. Route to provider
	provider, err := h.router.Get(req.Model)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported model"})
		return
	}

	// 3. Call provider
	if req.Stream {
		provider.Stream(c, userID, &req, h.quotaSvc, h.chatSave)
		return
	}

	requestAt := time.Now()
	resp, err := provider.Complete(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// 4. Deduct quota and save log async (Background avoids cancelled request context)
	reqMsgJSON, _ := json.Marshal(req.Messages)
	sessionID, _ := uuid.Parse(req.SessionID)
	go func() {
		ctx := context.Background()
		_ = h.quotaSvc.Deduct(ctx, userID, req.Model, resp.Usage.TotalTokens)
		_ = h.chatSave.Save(ctx, &domain.ChatLog{
			ID:              uuid.New(),
			UserID:          userID,
			SessionID:       sessionID,
			ModelID:         req.Model,
			RequestAt:       requestAt,
			ResponseAt:      time.Now(),
			RequestMessages: reqMsgJSON,
			ResponseContent: resp.Content,
			InputTokens:     resp.Usage.InputTokens,
			OutputTokens:    resp.Usage.OutputTokens,
			Status:          "success",
		})
	}()

	c.JSON(http.StatusOK, resp)
}
