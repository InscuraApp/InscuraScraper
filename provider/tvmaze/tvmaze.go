package tvmaze

import (
	"encoding/json"
	"fmt"
	"inscurascraper/provider"
	"inscurascraper/provider/internal/scraper"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"golang.org/x/text/language"
	"gorm.io/datatypes"
)

var (
	_ provider.MovieProvider = (*TVMaze)(nil)
	_ provider.MovieSearcher = (*TVMaze)(nil)
	_ provider.ActorProvider = (*TVMaze)(nil)
	_ provider.ActorSearcher = (*TVMaze)(nil)
	_ provider.ConfigSetter  = (*TVMaze)(nil)
)

const (
	Name     = "TVMaze"
	Priority = 1000
)

const (
	baseURL    = "https://www.tvmaze.com/"
	apiBaseURL = "https://api.tvmaze.com"
)

type TVMaze struct {
	*scraper.Scraper
	apiKey string
}

func New() *TVMaze {
	return &TVMaze{
		Scraper: scraper.NewDefaultScraper(
			Name, baseURL, Priority, language.English,
		),
	}
}

func (t *TVMaze) SetConfig(config provider.Config) error {
	if apiKey, err := config.GetString("api_key"); err == nil {
		t.apiKey = apiKey
	}
	return nil
}

// apiGet performs a GET request to the TVMaze API.
func (t *TVMaze) apiGet(apiURL string, dest any) error {
	c := t.ClonedCollector()

	var parseErr error
	c.OnResponse(func(r *colly.Response) {
		if err := json.Unmarshal(r.Body, dest); err != nil {
			parseErr = err
		}
	})

	headers := http.Header{}
	headers.Set("Accept", "application/json")
	if err := c.Request(http.MethodGet, apiURL, nil, nil, headers); err != nil {
		return err
	}
	return parseErr
}

// parseDate parses "2006-01-02" to datatypes.Date.
func parseDate(s string) datatypes.Date {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return datatypes.Date(t)
	}
	return datatypes.Date(time.Time{})
}

// stripHTML removes HTML tags from a string.
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	return strings.TrimSpace(htmlTagRe.ReplaceAllString(s, ""))
}

// parseIDFromURL extracts numeric ID from TVMaze URLs like
// https://www.tvmaze.com/shows/169/breaking-bad or
// https://www.tvmaze.com/people/14245/bryan-cranston
func parseIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(parts) < 2 {
		return "", provider.ErrInvalidURL
	}
	if _, err := strconv.Atoi(parts[1]); err != nil {
		return "", fmt.Errorf("invalid TVMaze ID: %s", parts[1])
	}
	return parts[1], nil
}

// Common response types.

type imageObj struct {
	Medium   string `json:"medium"`
	Original string `json:"original"`
}

type country struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	Timezone string `json:"timezone"`
}

type network struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Country      *country `json:"country"`
	OfficialSite *string  `json:"officialSite"`
}

type webChannel struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Country      *country `json:"country"`
	OfficialSite *string  `json:"officialSite"`
}

type rating struct {
	Average *float64 `json:"average"`
}

type externals struct {
	TvRage  *int    `json:"tvrage"`
	TheTVDB *int    `json:"thetvdb"`
	IMDB    *string `json:"imdb"`
}

type showImage struct {
	ID          int    `json:"id"`
	Type        string `json:"type"` // poster, background, banner, typography
	Main        bool   `json:"main"`
	Resolutions struct {
		Original *struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"original"`
		Medium *struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"medium"`
	} `json:"resolutions"`
}

func init() {
	provider.Register(Name, New)
}
