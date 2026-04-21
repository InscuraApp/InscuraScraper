package engine

import (
	"log"
	"os"

	"inscurascraper/common/fetch"
	mt "inscurascraper/provider"
)

func (e *Engine) init() *Engine {
	e.initLogger()
	e.initFetcher()
	e.initActorProviders()
	e.initMovieProviders()
	return e
}

func (e *Engine) initLogger() {
	e.logger = log.New(os.Stdout, "[ENGINE]\u0020", log.LstdFlags|log.Llongfile)
}

func (e *Engine) initFetcher() {
	e.fetcher = fetch.Default(&fetch.Config{Timeout: e.timeout})
}

// initActorProviders initializes actor providers.
func (e *Engine) initActorProviders() {
	for name, factory := range mt.RangeActorFactory {
		provider := factory()

		if s, ok := provider.(mt.RequestTimeoutSetter); ok {
			s.SetRequestTimeout(e.timeout /* global timeout */)
		}

		if config, hasConfig := e.actorProviderConfigs.Get(name); hasConfig {
			e.applyProviderConfig("actor", provider, config)
		}

		if provider.Priority() <= 0 {
			e.logger.Printf("Disable actor provider: %s", provider.Name())
			continue
		}

		// Add actor provider by name.
		e.actorProviders.Set(name, provider)
		// Add actor provider by host.
		host := provider.URL().Hostname()
		e.actorHostProviders.Set(host,
			append(e.actorHostProviders.
				GetOrDefault(host, nil), provider))
	}
}

// initMovieProviders initializes movie providers.
//
// A provider that implements both ActorProvider and MovieProvider is
// registered under the same Name() via two separate factories. To ensure
// per-request state (API keys, language, proxy) set via RequestConfig is
// visible regardless of whether the request lands on the actor or movie
// role, this function reuses the actor instance when one already exists
// with the same name. Otherwise a fresh instance is created.
func (e *Engine) initMovieProviders() {
	for name, factory := range mt.RangeMovieFactory {
		var provider mt.MovieProvider
		if existing, ok := e.actorProviders.Get(name); ok {
			if mp, ok := existing.(mt.MovieProvider); ok {
				provider = mp
			}
		}
		if provider == nil {
			provider = factory()

			if s, ok := provider.(mt.RequestTimeoutSetter); ok {
				s.SetRequestTimeout(e.timeout /* global timeout */)
			}

			if config, hasConfig := e.movieProviderConfigs.Get(name); hasConfig {
				e.applyProviderConfig("movie", provider, config)
			}
		}

		if provider.Priority() <= 0 {
			e.logger.Printf("Disable movie provider: %s", provider.Name())
			continue
		}

		// Add movie provider by name.
		e.movieProviders.Set(name, provider)
		// Add movie provider by host.
		host := provider.URL().Hostname()
		e.movieHostProviders.Set(host,
			append(e.movieHostProviders.
				GetOrDefault(host, nil), provider))
	}
}

func (e *Engine) applyProviderConfig(providerType string, provider mt.Provider, config mt.Config) {
	const (
		proxyConfigKey    = "proxy"
		priorityConfigKey = "priority"
		timeoutConfigKey  = "timeout"
	)

	// Apply overridden priority.
	if config.Has(priorityConfigKey) {
		if v, err := config.GetFloat64(priorityConfigKey); err == nil {
			e.logger.Printf("Override %s provider priority: %s=%.2f", providerType, provider.Name(), v)
			provider.SetPriority(v)
		}
	}

	// Apply overridden proxy.
	if s, ok := provider.(mt.ProxySetter); ok && config.Has(proxyConfigKey) {
		if v, err := config.GetString(proxyConfigKey); err == nil {
			if err := s.SetProxy(v); err != nil {
				e.logger.Fatalf("Set proxy for %s provider '%s' error: %v", providerType, provider.Name(), v)
			}
			e.providerProxies.Set(provider.Name(), v)
			e.logger.Printf("Override %s provider proxy: %s=%s", providerType, provider.Name(), sanitizeURL(v))
		}
	}

	// Apply request timeout.
	if s, ok := provider.(mt.RequestTimeoutSetter); ok && config.Has(timeoutConfigKey) {
		if v, err := config.GetDuration(timeoutConfigKey); err == nil {
			e.logger.Printf("Override %s provider request timeout: %s=%s", providerType, provider.Name(), v)
			s.SetRequestTimeout(v)
		}
	}

	// Apply full config.
	if s, ok := provider.(mt.ConfigSetter); ok {
		if err := s.SetConfig(config); err != nil {
			e.logger.Fatalf("Set %s provider config for %s: %v", providerType, provider.Name(), err)
		}
	}
}
