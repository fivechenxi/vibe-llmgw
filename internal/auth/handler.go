package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/llmgw/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	cfg *config.Config
	db  *pgxpool.Pool
}

func NewHandler(cfg *config.Config, db *pgxpool.Pool) *Handler {
	return &Handler{cfg: cfg, db: db}
}

// Login redirects the user to the SSO provider login page.
func (h *Handler) Login(c *gin.Context) {
	// TODO: build SSO redirect URL based on cfg.SSO.Provider
	c.Redirect(http.StatusFound, "/sso/placeholder")
}

// Callback handles the SSO provider callback, exchanges code for user info,
// upserts the user record, and issues a JWT.
func (h *Handler) Callback(c *gin.Context) {
	// TODO: exchange code → user info → upsert user → sign JWT
	c.JSON(http.StatusOK, gin.H{"token": "TODO"})
}

// Logout invalidates the client-side token (stateless; client drops the JWT).
func (h *Handler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}