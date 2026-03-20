package proxy

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/llmgw/internal/chat"
	"github.com/yourorg/llmgw/internal/config"
	"github.com/yourorg/llmgw/internal/domain"
	"github.com/yourorg/llmgw/internal/middleware"
	"github.com/yourorg/llmgw/internal/quota"
)

type Handler struct {
	cfg      *config.Config
	quotaSvc *quota.Service
	chatRepo *chat.Repository
	router   *Router
}

func NewHandler(cfg *config.Config, quotaSvc *quota.Service, chatRepo *chat.Repository) *Handler {
	return &Handler{
		cfg:      cfg,
		quotaSvc: quotaSvc,
		chatRepo: chatRepo,
		router:   NewRouter(cfg),
	}
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
		if err == quota.ErrQuotaExceeded {
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

	// 3. Call provider (streaming or not)
	if req.Stream {
		provider.Stream(c, userID, &req, h.quotaSvc, h.chatRepo)
		return
	}

	resp, err := provider.Complete(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// 4. Async deduct quota
	go h.quotaSvc.Deduct(c.Request.Context(), userID, req.Model, resp.Usage.TotalTokens)

	c.JSON(http.StatusOK, resp)
}