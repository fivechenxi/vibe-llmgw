package model

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/llmgw/internal/domain"
	"github.com/yourorg/llmgw/internal/middleware"
	"github.com/yourorg/llmgw/internal/quota"
)

// quotaLister is the narrow interface Handler needs from the quota repository.
// *quota.Repository satisfies it; tests can substitute a stub.
type quotaLister interface {
	ListByUser(ctx context.Context, userID string) ([]domain.UserQuota, error)
}

type Handler struct {
	repo      *Repository
	quotaRepo quotaLister
}

func NewHandler(repo *Repository, quotaRepo *quota.Repository) *Handler {
	return &Handler{repo: repo, quotaRepo: quotaRepo}
}

// ListModels returns active models that the current user has quota assigned for.
func (h *Handler) ListModels(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)

	quotas, err := h.quotaRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type modelWithQuota struct {
		ModelID   string `json:"model_id"`
		Remaining int64  `json:"remaining_tokens"`
	}
	result := make([]modelWithQuota, 0, len(quotas))
	for _, q := range quotas {
		if q.Remaining() > 0 {
			result = append(result, modelWithQuota{
				ModelID:   q.ModelID,
				Remaining: q.Remaining(),
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"models": result})
}

// ListQuota returns full quota details for the current user across all models.
func (h *Handler) ListQuota(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	quotas, err := h.quotaRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"quotas": quotas})
}