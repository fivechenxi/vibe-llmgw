package chat

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourorg/llmgw/internal/domain"
	"github.com/yourorg/llmgw/internal/middleware"
)

// chatRepo is the narrow interface Handler needs from the persistence layer.
// *Repository satisfies it; tests can substitute a stub.
type chatRepo interface {
	ListSessions(ctx context.Context, userID string) ([]uuid.UUID, error)
	GetSession(ctx context.Context, userID string, sessionID uuid.UUID) ([]domain.ChatLog, error)
}

type Handler struct {
	repo chatRepo
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) ListSessions(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	sessions, err := h.repo.ListSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

func (h *Handler) GetSession(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}

	logs, err := h.repo.GetSession(c.Request.Context(), userID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": logs})
}