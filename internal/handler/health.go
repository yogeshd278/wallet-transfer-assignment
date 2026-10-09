package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health responds with liveness; when DBPing is set, includes database readiness.
func Health(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload := gin.H{"status": "ok"}
		if deps.DBPing != nil {
			if err := deps.DBPing(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "degraded",
					"db":     "unavailable",
				})
				return
			}
			payload["db"] = "ok"
		}
		c.JSON(http.StatusOK, payload)
	}
}
