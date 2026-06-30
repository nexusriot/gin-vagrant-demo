package v1

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nexusriot/gin-vagrant-demo/internal/config"
	"github.com/nexusriot/gin-vagrant-demo/internal/middleware"
)

type AuthHandler struct {
	Secret  string
	Clients []config.Client
	Log     *slog.Logger
}

type tokenRequest struct {
	ClientID     string `json:"client_id"     binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
}

type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

const tokenTTL = 24 * time.Hour

func (h *AuthHandler) Token(c *gin.Context) {
	var req tokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.validClient(req.ClientID, req.ClientSecret) {
		h.Log.Warn("audit",
			"audit", true,
			"event", "auth_failed",
			"client_id", req.ClientID,
			"client_ip", c.ClientIP(),
			"request_id", c.GetString(middleware.RequestIDKey),
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   req.ClientID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	signed, err := tok.SignedString([]byte(h.Secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not issue token"})
		return
	}
	h.Log.Info("audit",
		"audit", true,
		"event", "auth_token_issued",
		"client_id", req.ClientID,
		"client_ip", c.ClientIP(),
		"request_id", c.GetString(middleware.RequestIDKey),
	)
	c.JSON(http.StatusOK, tokenResponse{Token: signed, ExpiresIn: int(tokenTTL.Seconds())})
}

func (h *AuthHandler) validClient(id, secret string) bool {
	for _, c := range h.Clients {
		if c.ID == id && c.Secret == secret {
			return true
		}
	}
	return false
}
