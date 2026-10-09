package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"
const requestIDContextKey = "request_id"

// RequestID ensures every request has a stable ID for logs and tracing.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(requestIDContextKey, id)
		c.Writer.Header().Set(requestIDHeader, id)
		c.Next()
	}
}

// RequestIDFromContext returns the request ID set by RequestID middleware.
func RequestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get(requestIDContextKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
