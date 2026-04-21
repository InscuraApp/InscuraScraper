package tvdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"golang.org/x/text/language"
	"gorm.io/datatypes"

	"inscurascraper/provider"
	"inscurascraper/provider/internal/scraper"
)

var (
	_ provider.MovieProvider = (*TVDB)(nil)
	_ provider.MovieSearcher = (*TVDB)(nil)
	_ provider.ActorProvider = (*TVDB)(nil)
	_ provider.ActorSearcher = (*TVDB)(nil)
	_ provider.ConfigSetter  = (*TVDB)(nil)
)

const (
	Name     = "TVDB"
	Priority = 1000
)

const (
	baseURL    = "https://www.thetvdb.com/"
	apiBaseURL = "https://api4.thetvdb.com/v4"

	seriesPageURL = "https://www.thetvdb.com/series/%s"
	moviePageURL  = "https://www.thetvdb.com/movies/%s"
	personPageURL = "https://www.thetvdb.com/people/%s"

	defaultLanguage = "zho" // Chinese
)

// ID prefixes to distinguish series from movies.
const (
	seriesPrefix = "series:"
	moviePrefix  = "movie:"
)

type TVDB struct {
	*scraper.Scraper
	apiKey      string
	token       string
	tokenExpiry time.Time
	cachedAPIKey string
	mu          sync.Mutex
}

func New() *TVDB {
	return &TVDB{
		Scraper: scraper.NewDefaultScraper(
			Name, baseURL, Priority, language.Chinese,
		),
	}
}

func (t *TVDB) SetConfig(config provider.Config) error {
	if apiKey, err := config.GetString("api_key"); err == nil {
		t.apiKey = apiKey
	}
	return nil
}

// resolveAPIKey returns the effective API key for the current request.
// A per-request X-Is-Api-Key-TVDB header takes precedence over the global config.
func (t *TVDB) resolveAPIKey() string {
	if k := t.GetRequestConfig().APIKeyFor(Name); k != "" {
		return k
	}
	return t.apiKey
}

// bcp47ToTVDBLang maps a BCP 47 tag (e.g. "zh-CN", "en-US") to the ISO 639-2/T
// 3-letter code that TVDB uses in translations[].language. Unknown tags fall
// through to defaultLanguage so existing behavior is preserved.
var bcp47ToTVDBLang = map[string]string{
	"zh": "zho", "en": "eng", "ja": "jpn", "ko": "kor",
	"fr": "fra", "de": "deu", "es": "spa", "it": "ita",
	"pt": "por", "ru": "rus", "ar": "ara", "he": "heb",
	"hi": "hin", "id": "ind", "th": "tha", "tr": "tur",
	"vi": "vie", "pl": "pol", "nl": "nld", "sv": "swe",
	"da": "dan", "no": "nor", "fi": "fin", "cs": "ces",
	"hu": "hun", "uk": "ukr", "el": "ell", "ro": "ron",
	"bg": "bul", "hr": "hrv", "sk": "slk", "sl": "slv",
	"lt": "lit", "lv": "lav", "et": "est", "ms": "msa",
	"fa": "fas", "bn": "ben", "ta": "tam", "te": "tel",
	"mr": "mar", "ur": "urd",
}

// resolveLanguage returns the TVDB 3-letter language code for the current request.
// Falls back to defaultLanguage ("zho") when the X-Is-Language header is not set
// or does not map to a known code.
func (t *TVDB) resolveLanguage() string {
	tag := t.GetRequestConfig().LanguageOr("")
	if tag == "" {
		return defaultLanguage
	}
	// Take the primary language subtag, e.g. "zh-CN" → "zh".
	base := strings.ToLower(tag)
	if i := strings.IndexAny(base, "-_"); i > 0 {
		base = base[:i]
	}
	if code, ok := bcp47ToTVDBLang[base]; ok {
		return code
	}
	return defaultLanguage
}

// login authenticates with the TVDB API and returns a valid JWT token.
// Token is cached and shared across requests using the same API key.
func (t *TVDB) login(apiKey string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Return cached token if still valid and for the same key.
	if t.token != "" && t.cachedAPIKey == apiKey &&
		time.Now().Before(t.tokenExpiry.Add(-24*time.Hour)) {
		return t.token, nil
	}

	body, _ := json.Marshal(map[string]string{"apikey": apiKey})

	c := t.ClonedCollector()

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
		Status string `json:"status"`
	}
	var parseErr error

	c.OnResponse(func(r *colly.Response) {
		if err := json.Unmarshal(r.Body, &loginResp); err != nil {
			parseErr = err
		}
	})

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", "application/json")
	if err := c.Request(http.MethodPost, apiBaseURL+"/login", bytes.NewReader(body), nil, headers); err != nil {
		return "", fmt.Errorf("tvdb login failed: %w", err)
	}
	if parseErr != nil {
		return "", fmt.Errorf("tvdb login parse failed: %w", parseErr)
	}
	if loginResp.Data.Token == "" {
		return "", fmt.Errorf("tvdb login returned empty token")
	}

	t.token = loginResp.Data.Token
	t.tokenExpiry = time.Now().Add(28 * 24 * time.Hour)
	t.cachedAPIKey = apiKey
	return t.token, nil
}

// apiGet performs an authenticated GET request to the TVDB API.
func (t *TVDB) apiGet(apiURL string, dest any) error {
	apiKey := t.resolveAPIKey()
	if apiKey == "" {
		return provider.ErrProviderNotFound
	}

	token, err := t.login(apiKey)
	if err != nil {
		return err
	}

	c := t.ClonedCollector()

	var parseErr error
	c.OnResponse(func(r *colly.Response) {
		if err := json.Unmarshal(r.Body, dest); err != nil {
			parseErr = err
		}
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	headers.Set("Accept", "application/json")
	if err := c.Request(http.MethodGet, apiURL, nil, nil, headers); err != nil {
		return err
	}
	return parseErr
}

// parseDate parses a TVDB date string "2006-01-02" to datatypes.Date.
func parseDate(s string) datatypes.Date {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return datatypes.Date(t)
	}
	return datatypes.Date(time.Time{})
}

// parseMovieID splits a prefixed ID like "series:12345" or "movie:12345".
// Returns (type, numericID). Defaults to "series" if no prefix.
func parseMovieID(id string) (kind string, numericID string) {
	if strings.HasPrefix(id, seriesPrefix) {
		return "series", strings.TrimPrefix(id, seriesPrefix)
	}
	if strings.HasPrefix(id, moviePrefix) {
		return "movie", strings.TrimPrefix(id, moviePrefix)
	}
	return "series", id
}

// parseSlugFromURL extracts the type and slug from a TVDB URL.
// e.g. "https://www.thetvdb.com/series/game-of-thrones" → ("series", "game-of-thrones")
func parseSlugFromURL(rawURL string) (kind, slug string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 2)
	if len(parts) < 2 || parts[1] == "" {
		return "", "", fmt.Errorf("invalid TVDB URL: %s", rawURL)
	}
	return parts[0], parts[1], nil
}

// parsePersonIDFromURL extracts numeric ID from "/people/12345-name".
func parsePersonIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	base := path.Base(u.Path)
	if idx := strings.Index(base, "-"); idx > 0 {
		base = base[:idx]
	}
	if _, err := strconv.Atoi(base); err != nil {
		return "", fmt.Errorf("invalid TVDB person ID: %s", base)
	}
	return base, nil
}

// findTranslation returns name and overview in the preferred language.
func findTranslation(translations []translation, lang string) (name, overview string) {
	for _, tr := range translations {
		if tr.Language == lang {
			return tr.Name, tr.Overview
		}
	}
	// Fallback to English.
	for _, tr := range translations {
		if tr.Language == "eng" {
			return tr.Name, tr.Overview
		}
	}
	return "", ""
}

// Common API response types.

type apiResponse[T any] struct {
	Data   T      `json:"data"`
	Status string `json:"status"`
}

type translation struct {
	Name       string `json:"name"`
	Overview   string `json:"overview"`
	Language   string `json:"language"`
	IsPrimary  bool   `json:"isPrimary"`
	TagLine    string `json:"tagline"`
}

type genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type character struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	PeopleID      int    `json:"peopleId"`
	PersonName    string `json:"personName"`
	PersonImgURL  string `json:"personImgURL"`
	Image         string `json:"image"`
	IsFeatured    bool   `json:"isFeatured"`
	Type          int    `json:"type"`
	Sort          int    `json:"sort"`
}

type artwork struct {
	ID        int    `json:"id"`
	Image     string `json:"image"`
	Thumbnail string `json:"thumbnail"`
	Type      int    `json:"type"`
	Language  string `json:"language"`
	Score     int    `json:"score"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type trailer struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Language string `json:"language"`
	Runtime  int    `json:"runtime"`
}

type company struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type remoteID struct {
	ID         string `json:"id"`
	Type       int    `json:"type"`
	SourceName string `json:"sourceName"`
}

type biography struct {
	Biography string `json:"biography"`
	Language  string `json:"language"`
}

func init() {
	provider.Register(Name, New)
}
