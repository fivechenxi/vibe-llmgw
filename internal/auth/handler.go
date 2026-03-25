package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/llmgw/internal/config"
	"github.com/yourorg/llmgw/internal/domain"
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

// DevLogin issues a JWT for a known user by username.
// Only available in non-production environments.
func (h *Handler) DevLogin(c *gin.Context) {
	if h.cfg.Env == "production" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user domain.User
	err := h.db.QueryRow(c.Request.Context(),
		"SELECT id, email, name FROM users WHERE id = $1", req.Username,
	).Scan(&user.ID, &user.Email, &user.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	expireHours := h.cfg.JWT.ExpireHours
	if expireHours == 0 {
		expireHours = 24
	}
	token, err := SignToken(&user, h.cfg.JWT.Secret, expireHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}