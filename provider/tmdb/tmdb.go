package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"golang.org/x/text/language"
	"gorm.io/datatypes"

	"inscurascraper/provider"
	"inscurascraper/provider/internal/scraper"
)

var (
	_ provider.MovieProvider = (*TMDB)(nil)
	_ provider.MovieSearcher = (*TMDB)(nil)
	_ provider.ActorProvider = (*TMDB)(nil)
	_ provider.ActorSearcher = (*TMDB)(nil)
	_ provider.ConfigSetter  = (*TMDB)(nil)
)

const (
	Name     = "TMDB"
	Priority = 1000
)

const (
	baseURL      = "https://www.themoviedb.org/"
	apiBaseURL   = "https://api.themoviedb.org/3"
	imageBaseURL = "https://image.tmdb.org/t/p/"

	moviePageURL  = "https://www.themoviedb.org/movie/%s"
	personPageURL = "https://www.themoviedb.org/person/%s"

	defaultLanguage = "zh-CN"
)

// Image sizes
const (
	posterW342      = "w342"
	posterW500      = "w500"
	posterOriginal  = "original"
	backdropW780    = "w780"
	backdropOriginal = "original"
	profileOriginal = "original"
)

type TMDB struct {
	*scraper.Scraper
	apiToken string
}

func New() *TMDB {
	return &TMDB{
		Scraper: scraper.NewDefaultScraper(
			Name, baseURL, Priority, language.Chinese,
		),
	}
}

func (t *TMDB) SetConfig(config provider.Config) error {
	if apiToken, err := config.GetString("api_token"); err == nil {
		t.apiToken = apiToken
	}
	return nil
}

// imageURL builds a full TMDB image URL.
func imageURL(size, filePath string) string {
	if filePath == "" {
		return ""
	}
	return imageBaseURL + size + filePath
}

// parseDate parses a TMDB date string "2006-01-02" to datatypes.Date.
func parseDate(s string) datatypes.Date {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return datatypes.Date(t)
	}
	return datatypes.Date(time.Time{})
}

// parseIDFromPath extracts the numeric ID from a TMDB URL path segment
// like "550-fight-club" → "550" or "550" → "550".
func parseIDFromPath(rawURL string) (string, error) {
	homepage, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	base := path.Base(homepage.Path)
	if idx := strings.Index(base, "-"); idx > 0 {
		base = base[:idx]
	}
	if _, err := strconv.Atoi(base); err != nil {
		return "", fmt.Errorf("invalid TMDB ID: %s", base)
	}
	return base, nil
}

// resolveAPIToken returns the effective API token for the current request.
// A per-request X-Is-Api-Key-TMDB header takes precedence over the global config.
func (t *TMDB) resolveAPIToken() string {
	if k := t.GetRequestConfig().APIKeyFor(Name); k != "" {
		return k
	}
	return t.apiToken
}

// resolveLanguage returns the BCP 47 tag to send as TMDB ?language=.
// Falls back to defaultLanguage when the X-Is-Language header is not set.
// TMDB accepts BCP 47 tags directly (e.g. "zh-CN", "en-US", "ja-JP").
func (t *TMDB) resolveLanguage() string {
	return t.GetRequestConfig().LanguageOr(defaultLanguage)
}

// apiGet performs a GET request to the TMDB API with Bearer auth,
// parses the JSON response into dest.
func (t *TMDB) apiGet(apiURL string, dest any) error {
	token := t.resolveAPIToken()
	if token == "" {
		return provider.ErrProviderNotFound
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

func init() {
	provider.Register(Name, New)
}
