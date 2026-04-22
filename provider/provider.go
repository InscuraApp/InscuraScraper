// Package provider defines the extensibility contract for InscuraScraper's
// metadata sources. A provider implements one or more of the interfaces
// below (most commonly [MovieProvider] and/or [ActorProvider]) and registers
// itself with [Register] during package init. The engine then discovers the
// provider by blank-importing its package from engine/register.go.
//
// Interface overview:
//
//   - [Provider]        — base interface every provider must implement.
//   - [MovieProvider]   — retrieve movie info by ID or URL.
//   - [ActorProvider]   — retrieve actor info by ID or URL.
//   - [MovieSearcher]   — optional; enables movie search.
//   - [ActorSearcher]   — optional; enables actor search.
//   - [MovieReviewer]   — optional; returns user reviews for a movie.
//   - [Fetcher]         — optional; custom HTTP fetch (referer, cookies…).
//   - [ConfigSetter]    — optional; accepts per-provider configuration.
//   - [ProxySetter]     — optional; applies an HTTP/SOCKS5 proxy.
//   - [RequestTimeoutSetter] — optional; adjusts request timeout.
//
// See CLAUDE.md for the full provider-development guide.
package provider

import (
	"inscurascraper/model"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/text/language"
)

// Provider is the base interface implemented by every metadata source.
// It exposes identity (Name, URL), language, and the priority used by the
// engine to rank results when multiple providers match.
type Provider interface {
	// Name returns the name of the provider.
	Name() string

	// Priority returns the matching priority of the provider.
	Priority() float64

	// SetPriority sets the provider priority to the given value.
	SetPriority(v float64)

	// Language returns the primary language supported by the provider.
	Language() language.Tag

	// URL returns the base url of the provider.
	URL() *url.URL
}

// MovieSearcher is implemented by providers that can search for movies by
// keyword. Providers that only support direct ID/URL lookup should omit it.
type MovieSearcher interface {
	// SearchMovie searches matched movies.
	SearchMovie(keyword string) ([]*model.MovieSearchResult, error)

	// NormalizeMovieKeyword converts movie keyword to provider-friendly form.
	NormalizeMovieKeyword(Keyword string) string
}

// MovieReviewer is implemented by providers that expose user reviews or
// ratings for a movie. The engine skips review retrieval for providers that
// do not implement this interface.
type MovieReviewer interface {
	// GetMovieReviewsByID gets the user reviews of given movie id.
	GetMovieReviewsByID(id string) ([]*model.MovieReviewDetail, error)

	// GetMovieReviewsByURL gets the user reviews of given movie URL.
	GetMovieReviewsByURL(rawURL string) ([]*model.MovieReviewDetail, error)
}

// MovieProvider is the required interface for a provider that returns
// movie metadata. Implementations must also satisfy [Provider].
type MovieProvider interface {
	// Provider should be implemented.
	Provider

	// NormalizeMovieID normalizes movie ID to conform to standard.
	NormalizeMovieID(id string) string

	// ParseMovieIDFromURL parses movie ID from given URL.
	ParseMovieIDFromURL(rawURL string) (string, error)

	// GetMovieInfoByID gets movie's info by id.
	GetMovieInfoByID(id string) (*model.MovieInfo, error)

	// GetMovieInfoByURL gets movie's info by url.
	GetMovieInfoByURL(url string) (*model.MovieInfo, error)
}

// ActorSearcher is implemented by providers that can search for actors
// by keyword.
type ActorSearcher interface {
	// SearchActor searches matched actor/s.
	SearchActor(keyword string) ([]*model.ActorSearchResult, error)
}

// ActorProvider is the required interface for a provider that returns
// actor metadata. Implementations must also satisfy [Provider].
type ActorProvider interface {
	// Provider should be implemented.
	Provider

	// NormalizeActorID normalizes actor ID to conform to standard.
	NormalizeActorID(id string) string

	// ParseActorIDFromURL parses actor ID from given URL.
	ParseActorIDFromURL(rawURL string) (string, error)

	// GetActorInfoByID gets actor's info by id.
	GetActorInfoByID(id string) (*model.ActorInfo, error)

	// GetActorInfoByURL gets actor's info by url.
	GetActorInfoByURL(url string) (*model.ActorInfo, error)
}

// Fetcher is implemented by providers that need to serve media resources
// (images, previews) through a customized HTTP client — for example one
// that sets a Referer header, maintains cookies, or handles age-gates.
// The engine delegates outbound media fetching to this method when present.
type Fetcher interface {
	// Fetch fetches media resources from url.
	Fetch(url string) (*http.Response, error)
}

// RequestTimeoutSetter lets the engine apply a custom per-request timeout
// at initialization, overriding the provider's default.
type RequestTimeoutSetter interface {
	// SetRequestTimeout sets timeout for HTTP requests.
	SetRequestTimeout(timeout time.Duration)
}

// ProxySetter lets the engine apply an HTTP or SOCKS5 proxy to the
// provider's outbound HTTP client.
type ProxySetter interface {
	// SetProxy sets http or socks5 proxy for HTTP requests.
	SetProxy(proxyURL string) error
}

// RequestConfig holds per-request overrides for proxy, per-provider API keys,
// the preferred response language, and a parsed search year (extracted from
// release-name filenames).
// Stored per-goroutine in the Scraper to support concurrent multi-user requests.
//
// Proxy is global for the request (all providers share one outbound proxy).
// APIKeys is keyed by provider name (lower-cased) so a single request can
// carry different keys for different providers (e.g. TMDB + TVDB fanout).
// Language is a BCP 47 tag (e.g. "zh-CN", "en-US"); each provider maps it
// to its own format in resolveLanguage().
// SearchYear is the release year parsed out of the raw search keyword by
// the engine (from names like "Spider-Man.No.Way.Home.2021.1080p..."), so
// providers that support year filtering (e.g. TMDB's &year=) can narrow
// their match without the provider having to re-parse the filename.
type RequestConfig struct {
	Proxy      string
	APIKeys    map[string]string
	Language   string
	SearchYear int
}

// APIKeyFor returns the per-request API key for the given provider name
// (case-insensitive). Returns "" when no override applies, including when
// the receiver is nil — callers can safely chain without nil checks.
func (c *RequestConfig) APIKeyFor(providerName string) string {
	if c == nil || len(c.APIKeys) == 0 {
		return ""
	}
	return c.APIKeys[strings.ToLower(providerName)]
}

// LanguageOr returns the per-request language tag, or fallback when unset.
// Nil-safe — callers can chain without checking the receiver.
func (c *RequestConfig) LanguageOr(fallback string) string {
	if c == nil || c.Language == "" {
		return fallback
	}
	return c.Language
}

// SearchYearOr returns the engine-parsed search year, or fallback when unset.
// Nil-safe.
func (c *RequestConfig) SearchYearOr(fallback int) int {
	if c == nil || c.SearchYear == 0 {
		return fallback
	}
	return c.SearchYear
}

// RequestConfigAccessor is implemented by scrapers that store a
// per-goroutine [RequestConfig]. The engine sets the config at the start of
// a request and clears it when done, allowing per-request overrides to flow
// through provider code without changing method signatures.
type RequestConfigAccessor interface {
	// SetRequestConfig stores a per-goroutine request config.
	SetRequestConfig(cfg *RequestConfig)
	// ClearRequestConfig removes the per-goroutine request config.
	ClearRequestConfig()
	// GetRequestConfig returns the current goroutine's request config (nil if none).
	GetRequestConfig() *RequestConfig
}

// Config is a read-only, type-safe accessor for provider configuration
// values sourced from environment variables (prefix IS_PROVIDER_<NAME>__).
// Keys are case-insensitive.
type Config interface {
	Has(string) bool
	GetString(string) (string, error)
	GetBool(string) (bool, error)
	GetInt64(string) (int64, error)
	GetFloat64(string) (float64, error)
	GetDuration(string) (time.Duration, error)
}

// ConfigSetter is implemented by providers that consume external
// configuration (API tokens, base URLs, feature flags). The engine invokes
// SetConfig during initialization with the provider-scoped [Config].
type ConfigSetter interface {
	// SetConfig sets any additional configs for Provider.
	SetConfig(config Config) error
}
