package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const SubjectKey = "subject"

// Auth validates a JWT Bearer token and sets SubjectKey in the Gin context.
func Auth(secret string, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			auditEvent(c, log, "auth_missing", "")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}
		raw := strings.TrimPrefix(header, "Bearer ")
		tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !tok.Valid {
			auditEvent(c, log, "auth_invalid", "")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		claims, ok := tok.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}
		sub, _ := claims.GetSubject()
		c.Set(SubjectKey, sub)
		c.Next()
	}
}

// AuditLog emits a structured audit entry for a successful privileged action.
func AuditLog(c *gin.Context, log *slog.Logger, event string) {
	auditEvent(c, log, event, c.GetString(SubjectKey))
}

func auditEvent(c *gin.Context, log *slog.Logger, event, subject string) {
	log.Warn("audit",
		"audit", true,
		"event", event,
		"subject", subject,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"client_ip", c.ClientIP(),
		"request_id", c.GetString(RequestIDKey),
	)
}
