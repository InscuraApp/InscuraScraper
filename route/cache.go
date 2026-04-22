package route

import (
	"github.com/gin-gonic/gin"
	cachecontrol "go.eigsys.de/gin-cachecontrol/v2"
)

func cacheNoStore() gin.HandlerFunc {
	return cachecontrol.New(cachecontrol.Config{
		// The no-store response directive indicates that any
		// caches of any kind (private or shared) should not
		// store this response.
		NoStore: true,
	})
}
