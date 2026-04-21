package scraper

import (
	"bytes"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"go.uber.org/atomic"
	"golang.org/x/text/language"

	"inscurascraper/provider"
)

var (
	_ provider.Provider              = (*Scraper)(nil)
	_ provider.ProxySetter           = (*Scraper)(nil)
	_ provider.RequestConfigAccessor = (*Scraper)(nil)
	_ provider.RequestTimeoutSetter  = (*Scraper)(nil)
)

// Scraper implements the basic Provider interface.
type Scraper struct {
	name     string
	baseURL  *url.URL
	priority *atomic.Float64
	language language.Tag
	c        *colly.Collector
	// Per-goroutine request configs: goroutine ID → *RequestConfig.
	requestConfigs sync.Map
}

// NewScraper returns a *Scraper that implements provider.Provider.
func NewScraper(name, base string, priority float64, lang language.Tag, opts ...Option) *Scraper {
	baseURL, err := url.Parse(base)
	if err != nil {
		panic(err)
	}
	s := &Scraper{
		name:     name,
		baseURL:  baseURL,
		priority: atomic.NewFloat64(priority),
		language: lang,
		c:        colly.NewCollector(),
	}
	for _, opt := range opts {
		// Apply options.
		if err := opt(s); err != nil {
			panic(err)
		}
	}
	return s
}

// NewDefaultScraper returns a *Scraper with default options enabled.
func NewDefaultScraper(name, baseURL string, priority float64, lang language.Tag, opts ...Option) *Scraper {
	return NewScraper(name, baseURL, priority, lang, append([]Option{
		WithAllowURLRevisit(),
		WithIgnoreRobotsTxt(),
		WithRandomUserAgent(),
	}, opts...)...)
}

func (s *Scraper) Name() string { return s.name }

func (s *Scraper) URL() *url.URL { return s.baseURL }

func (s *Scraper) Priority() float64 { return s.priority.Load() }

func (s *Scraper) SetPriority(v float64) { s.priority.Store(v) }

func (s *Scraper) Language() language.Tag { return s.language }

func (s *Scraper) NormalizeMovieID(id string) string { return id /* AS IS */ }

func (s *Scraper) ParseMovieIDFromURL(string) (string, error) { panic("unimplemented") }

func (s *Scraper) NormalizeActorID(id string) string { return id /* AS IS */ }

func (s *Scraper) ParseActorIDFromURL(string) (string, error) { panic("unimplemented") }

// ClonedCollector returns cloned internal collector.
// If the current goroutine has a request-scoped config with proxy,
// it is applied to the clone automatically.
func (s *Scraper) ClonedCollector() *colly.Collector {
	c := s.c.Clone()
	if cfg := s.GetRequestConfig(); cfg != nil && cfg.Proxy != "" {
		c.SetProxy(cfg.Proxy)
	}
	return c
}

// SetProxy sets http or socks5 proxy for HTTP requests (persistent/global).
func (s *Scraper) SetProxy(proxyURL string) error { return s.c.SetProxy(proxyURL) }

// SetRequestConfig stores a per-goroutine request config.
func (s *Scraper) SetRequestConfig(cfg *provider.RequestConfig) {
	s.requestConfigs.Store(goroutineID(), cfg)
}

// ClearRequestConfig removes the per-goroutine request config.
func (s *Scraper) ClearRequestConfig() {
	s.requestConfigs.Delete(goroutineID())
}

// GetRequestConfig returns the current goroutine's request config (nil if none).
func (s *Scraper) GetRequestConfig() *provider.RequestConfig {
	if v, ok := s.requestConfigs.Load(goroutineID()); ok {
		return v.(*provider.RequestConfig)
	}
	return nil
}

// SetRequestTimeout sets timeout for HTTP requests.
func (s *Scraper) SetRequestTimeout(timeout time.Duration) { s.c.SetRequestTimeout(timeout) }

// goroutineID returns the current goroutine's numeric ID.
// The format "goroutine NNN [..." is guaranteed by the Go runtime.
func goroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// Skip "goroutine " prefix (10 bytes), read digits until space.
	b := buf[10:n]
	if i := bytes.IndexByte(b, ' '); i > 0 {
		b = b[:i]
	}
	id, err := strconv.ParseUint(string(b), 10, 64)
	if err != nil {
		// Fallback: should never happen with standard Go runtime.
		return 0
	}
	return id
}
