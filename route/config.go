package route

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"inscurascraper/engine"
)

// getProxy returns the current persistent proxy configuration (read-only).
// Global proxy changes are only allowed via environment variables at startup.
// Per-request proxy is set via ?proxy= query parameter on each endpoint.
func getProxy(app *engine.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, &responseMessage{
			Data: app.GetProviderProxies(),
		})
	}
}
