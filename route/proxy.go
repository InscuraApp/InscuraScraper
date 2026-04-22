package route

import (
	"inscurascraper/common/proxy"
	"inscurascraper/engine"
	"net/http"
	"strings"

	mt "inscurascraper/provider"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

const (
	// headerProxy sets the per-request outbound proxy applied to every provider.
	headerProxy = "X-Is-Proxy"
	// headerAPIKeyPrefix prefixes per-provider API key headers, e.g.
	// "X-Is-Api-Key-TMDB". The provider name segment is case-insensitive.
	headerAPIKeyPrefix = "X-Is-Api-Key-"
	// headerLanguage sets the preferred response language (BCP 47 tag).
	// Each provider maps this to its own format.
	headerLanguage = "X-Is-Language"
)

// requestConfig is a middleware that extracts per-request overrides from
// HTTP headers and applies them as a per-goroutine RequestConfig on all
// providers for the duration of the request. Each concurrent request can
// use its own config safely.
//
// Headers:
//   - X-Is-Proxy:               proxy URL for all providers' outbound requests
//   - X-Is-Api-Key-{PROVIDER}:  API key for the named provider (case-insensitive)
//   - X-Is-Language:            preferred language as a BCP 47 tag (e.g. "zh-CN")
func requestConfig(app *engine.Engine) gin.HandlerFunc {
	canonicalPrefix := http.CanonicalHeaderKey(headerAPIKeyPrefix)

	return func(c *gin.Context) {
		proxyURL := c.GetHeader(headerProxy)
		if proxyURL != "" {
			if err := proxy.ValidateProxyURL(proxyURL); err != nil {
				abortWithStatusMessage(c, http.StatusBadRequest, err.Error())
				return
			}
		}

		langTag := strings.TrimSpace(c.GetHeader(headerLanguage))
		if langTag != "" {
			if _, err := language.Parse(langTag); err != nil {
				abortWithStatusMessage(c, http.StatusBadRequest,
					"invalid "+headerLanguage+": "+err.Error())
				return
			}
		}

		var apiKeys map[string]string
		for name, values := range c.Request.Header {
			if len(values) == 0 || values[0] == "" {
				continue
			}
			canonical := http.CanonicalHeaderKey(name)
			if !strings.HasPrefix(canonical, canonicalPrefix) ||
				len(canonical) == len(canonicalPrefix) {
				continue
			}
			providerName := strings.ToLower(canonical[len(canonicalPrefix):])
			if apiKeys == nil {
				apiKeys = make(map[string]string)
			}
			apiKeys[providerName] = values[0]
		}

		if proxyURL != "" || langTag != "" || len(apiKeys) > 0 {
			app.SetRequestConfig(&mt.RequestConfig{
				Proxy:    proxyURL,
				APIKeys:  apiKeys,
				Language: langTag,
			})
			defer app.ClearRequestConfig()
		}
		c.Next()
	}
}
