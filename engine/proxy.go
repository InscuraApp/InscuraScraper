package engine

import (
	"fmt"
	gomaps "maps"
	"net/url"

	mt "inscurascraper/provider"
)

// sanitizeURL removes credentials from a URL for safe logging.
func sanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = url.User("***")
	return u.String()
}

// setProxyForProvider applies proxy to a single provider if it implements ProxySetter.
func (e *Engine) setProxyForProvider(provider mt.Provider, proxyURL string) error {
	s, ok := provider.(mt.ProxySetter)
	if !ok {
		return fmt.Errorf("provider %s does not support proxy", provider.Name())
	}
	return s.SetProxy(proxyURL)
}

// SetProviderProxy sets the persistent proxy for a specific provider (both actor and movie roles).
// Pass empty string to clear the proxy.
func (e *Engine) SetProviderProxy(name, proxyURL string) error {
	var found bool
	var lastErr error

	if provider, ok := e.actorProviders.Get(name); ok {
		found = true
		if err := e.setProxyForProvider(provider, proxyURL); err != nil {
			lastErr = err
		}
	}

	if provider, ok := e.movieProviders.Get(name); ok {
		found = true
		if err := e.setProxyForProvider(provider, proxyURL); err != nil {
			lastErr = err
		}
	}

	if !found {
		return mt.ErrProviderNotFound
	}

	if lastErr != nil {
		return lastErr
	}

	if proxyURL != "" {
		e.providerProxies.Set(name, proxyURL)
	} else {
		e.providerProxies.Delete(name)
	}
	e.logger.Printf("Set provider proxy: %s=%s", name, sanitizeURL(proxyURL))
	return nil
}

// SetGlobalProxy sets the same persistent proxy for all providers.
// Pass empty string to clear all proxies.
func (e *Engine) SetGlobalProxy(proxyURL string) error {
	seen := make(map[string]bool)

	for name, provider := range e.actorProviders.Iterator() {
		if seen[name] {
			continue
		}
		seen[name] = true
		if err := e.setProxyForProvider(provider, proxyURL); err != nil {
			e.logger.Printf("Failed to set proxy for actor provider %s: %v", name, err)
		}
	}

	for name, provider := range e.movieProviders.Iterator() {
		if seen[name] {
			continue
		}
		seen[name] = true
		if err := e.setProxyForProvider(provider, proxyURL); err != nil {
			e.logger.Printf("Failed to set proxy for movie provider %s: %v", name, err)
		}
	}

	if proxyURL != "" {
		for name := range seen {
			e.providerProxies.Set(name, proxyURL)
		}
	} else {
		for name := range seen {
			e.providerProxies.Delete(name)
		}
	}

	e.logger.Printf("Set global proxy: %s", sanitizeURL(proxyURL))
	return nil
}

// GetProviderProxies returns a copy of the current persistent proxy configuration.
func (e *Engine) GetProviderProxies() map[string]string {
	return gomaps.Collect(e.providerProxies.Iterator())
}

// SetRequestConfig sets a per-goroutine RequestConfig on ALL providers.
// Goroutine-safe: each goroutine's config is independent.
// Must call ClearRequestConfig when the request is done.
func (e *Engine) SetRequestConfig(cfg *mt.RequestConfig) {
	seen := make(map[string]bool)
	for name, provider := range e.actorProviders.Iterator() {
		if seen[name] {
			continue
		}
		seen[name] = true
		if s, ok := provider.(mt.RequestConfigAccessor); ok {
			s.SetRequestConfig(cfg)
		}
	}
	for name, provider := range e.movieProviders.Iterator() {
		if seen[name] {
			continue
		}
		seen[name] = true
		if s, ok := provider.(mt.RequestConfigAccessor); ok {
			s.SetRequestConfig(cfg)
		}
	}
}

// ClearRequestConfig clears the per-goroutine RequestConfig on ALL providers.
func (e *Engine) ClearRequestConfig() {
	seen := make(map[string]bool)
	for name, provider := range e.actorProviders.Iterator() {
		if seen[name] {
			continue
		}
		seen[name] = true
		if s, ok := provider.(mt.RequestConfigAccessor); ok {
			s.ClearRequestConfig()
		}
	}
	for name, provider := range e.movieProviders.Iterator() {
		if seen[name] {
			continue
		}
		seen[name] = true
		if s, ok := provider.(mt.RequestConfigAccessor); ok {
			s.ClearRequestConfig()
		}
	}
}

// SetRequestConfigForProvider sets per-goroutine config on a specific provider.
func (e *Engine) SetRequestConfigForProvider(provider mt.Provider, cfg *mt.RequestConfig) {
	if s, ok := provider.(mt.RequestConfigAccessor); ok {
		s.SetRequestConfig(cfg)
	}
}

// ClearRequestConfigForProvider clears per-goroutine config on a specific provider.
func (e *Engine) ClearRequestConfigForProvider(provider mt.Provider) {
	if s, ok := provider.(mt.RequestConfigAccessor); ok {
		s.ClearRequestConfig()
	}
}

// CaptureRequestConfig reads the current goroutine's RequestConfig from any provider.
// Used to capture the config before spawning child goroutines.
func (e *Engine) CaptureRequestConfig() *mt.RequestConfig {
	for _, provider := range e.actorProviders.Iterator() {
		if s, ok := provider.(mt.RequestConfigAccessor); ok {
			if cfg := s.GetRequestConfig(); cfg != nil {
				return cfg
			}
		}
	}
	for _, provider := range e.movieProviders.Iterator() {
		if s, ok := provider.(mt.RequestConfigAccessor); ok {
			if cfg := s.GetRequestConfig(); cfg != nil {
				return cfg
			}
		}
	}
	return nil
}
