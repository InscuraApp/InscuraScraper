package route

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"inscurascraper/engine"
)

// getHealthz returns a liveness probe that always succeeds once the HTTP
// server is accepting connections. Use this for container/orchestrator
// liveness checks — it does not probe downstream dependencies.
func getHealthz() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// getReadyz returns a readiness probe that verifies the engine's backing
// dependencies (database) are reachable. It returns 503 when the database
// cannot be queried, which signals orchestrators to stop routing traffic
// to this instance.
func getReadyz(app *engine.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := app.DBVersion(); err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"error":  err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
