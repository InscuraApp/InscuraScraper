package engine

import (
	"fmt"
	"inscurascraper/collection/maps"
	"inscurascraper/common/fetch"
	"inscurascraper/database"
	"log"
	gomaps "maps"
	"net/http"
	"net/url"
	"strings"
	"time"

	mt "inscurascraper/provider"

	"gorm.io/gorm"
)

const (
	DefaultEngineName     = "inscurascraper"
	DefaultRequestTimeout = time.Minute
)

type Engine struct {
	db      *gorm.DB
	name    string
	timeout time.Duration
	fetcher *fetch.Fetcher
	// Engine Logger
	logger *log.Logger
	// Proxy tracking: provider name → proxy URL (empty = no proxy)
	providerProxies *maps.CaseInsensitiveMap[string]
	// Name:Config Case-Insensitive Map
	actorProviderConfigs *maps.CaseInsensitiveMap[mt.Config]
	movieProviderConfigs *maps.CaseInsensitiveMap[mt.Config]
	// Name:Provider Case-Insensitive Map
	actorProviders *maps.CaseInsensitiveMap[mt.ActorProvider]
	movieProviders *maps.CaseInsensitiveMap[mt.MovieProvider]
	// Host:[]Provider Case-Insensitive Map
	// We need a []mt.ActorProvider here because sometimes providers
	// can share the same host, but they're two different providers.
	// However, in most cases, a host is mapped to only one provider.
	// E.g., github.com -> [Gfriends, ...]
	actorHostProviders *maps.CaseInsensitiveMap[[]mt.ActorProvider]
	movieHostProviders *maps.CaseInsensitiveMap[[]mt.MovieProvider]
}

func New(db *gorm.DB, opts ...Option) *Engine {
	engine := &Engine{
		db:      db,
		name:    DefaultEngineName,
		timeout: DefaultRequestTimeout,
		// pre-initialize case-insensitive maps.
		providerProxies:      maps.NewCaseInsensitiveMap[string](),
		actorProviderConfigs: maps.NewCaseInsensitiveMap[mt.Config](),
		movieProviderConfigs: maps.NewCaseInsensitiveMap[mt.Config](),
		actorProviders:       maps.NewCaseInsensitiveMap[mt.ActorProvider](),
		movieProviders:       maps.NewCaseInsensitiveMap[mt.MovieProvider](),
		actorHostProviders:   maps.NewCaseInsensitiveMap[[]mt.ActorProvider](),
		movieHostProviders:   maps.NewCaseInsensitiveMap[[]mt.MovieProvider](),
	}
	// apply options.
	for _, opt := range opts {
		opt(engine)
	}
	return engine.init()
}

func Default() *Engine {
	db, _ := database.Open(&database.Config{
		DSN:                  "",
		DisableAutomaticPing: true,
	})
	engine := New(db)
	defer engine.DBAutoMigrate(true)
	return engine
}

func (e *Engine) IsActorProvider(name string) bool {
	return e.actorProviders.Has(name)
}

func (e *Engine) GetActorProviders() map[string]mt.ActorProvider {
	return gomaps.Collect(e.actorProviders.Iterator())
}

func (e *Engine) GetActorProviderByURL(rawURL string) (mt.ActorProvider, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	for _, p := range e.actorHostProviders.GetOrDefault(u.Hostname(), nil) {
		if strings.HasPrefix(u.Path, p.URL().Path) {
			return p, nil
		}
	}
	return nil, mt.ErrProviderNotFound
}

func (e *Engine) GetActorProviderByName(name string) (mt.ActorProvider, error) {
	provider, ok := e.actorProviders.Get(name)
	if !ok {
		return nil, mt.ErrProviderNotFound
	}
	return provider, nil
}

func (e *Engine) MustGetActorProviderByName(name string) mt.ActorProvider {
	provider, err := e.GetActorProviderByName(name)
	if err != nil {
		panic(err)
	}
	return provider
}

func (e *Engine) IsMovieProvider(name string) bool {
	return e.movieProviders.Has(name)
}

func (e *Engine) GetMovieProviders() map[string]mt.MovieProvider {
	return gomaps.Collect(e.movieProviders.Iterator())
}

func (e *Engine) GetMovieProviderByURL(rawURL string) (mt.MovieProvider, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	for _, p := range e.movieHostProviders.GetOrDefault(u.Hostname(), nil) {
		if strings.HasPrefix(u.Path, p.URL().Path) {
			return p, nil
		}
	}
	return nil, mt.ErrProviderNotFound
}

func (e *Engine) GetMovieProviderByName(name string) (mt.MovieProvider, error) {
	provider, ok := e.movieProviders.Get(name)
	if !ok {
		return nil, mt.ErrProviderNotFound
	}
	return provider, nil
}

func (e *Engine) MustGetMovieProviderByName(name string) mt.MovieProvider {
	provider, err := e.GetMovieProviderByName(name)
	if err != nil {
		panic(err)
	}
	return provider
}

// Fetch fetches content from url. If the provider
// is nil, the default fetcher will be used.
func (e *Engine) Fetch(url string, provider mt.Provider) (*http.Response, error) {
	// Provider which implements Fetcher interface should be
	// used to fetch all its corresponding resources.
	if fetcher, ok := provider.(mt.Fetcher); ok {
		return fetcher.Fetch(url)
	}
	return e.fetcher.Fetch(url)
}

// String returns the name of the Engine instance.
func (e *Engine) String() string { return e.name }

// movieProviderPriority returns a provider's Priority by name (0 when
// unknown). Designed for the engine/rank package which doesn't depend on
// engine-internal types.
func (e *Engine) movieProviderPriority(name string) float64 {
	if p, ok := e.movieProviders.Get(name); ok {
		return p.Priority()
	}
	return 0
}

// actorProviderPriority returns an actor provider's Priority by name.
func (e *Engine) actorProviderPriority(name string) float64 {
	if p, ok := e.actorProviders.Get(name); ok {
		return p.Priority()
	}
	return 0
}

// maxMovieProviderPriority returns the largest Priority across registered
// movie providers; used to normalise rank weights.
func (e *Engine) maxMovieProviderPriority() float64 {
	var max float64
	for _, p := range e.movieProviders.Iterator() {
		if v := p.Priority(); v > max {
			max = v
		}
	}
	return max
}

// maxActorProviderPriority returns the largest Priority across registered
// actor providers.
func (e *Engine) maxActorProviderPriority() float64 {
	var max float64
	for _, p := range e.actorProviders.Iterator() {
		if v := p.Priority(); v > max {
			max = v
		}
	}
	return max
}

var (
	_ = New
	_ = Default
)

var _ fmt.Stringer = (*Engine)(nil)
