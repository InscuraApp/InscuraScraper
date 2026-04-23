package trakt

import (
	"encoding/json"
	"inscurascraper/provider"
	"inscurascraper/provider/internal/scraper"
	"net/http"
	"time"

	"github.com/gocolly/colly/v2"
	"golang.org/x/text/language"
	"gorm.io/datatypes"
)

var (
	_ provider.MovieProvider = (*Trakt)(nil)
	_ provider.MovieSearcher = (*Trakt)(nil)
	_ provider.ActorProvider = (*Trakt)(nil)
	_ provider.ActorSearcher = (*Trakt)(nil)
	_ provider.ConfigSetter  = (*Trakt)(nil)
)

const (
	Name     = "Trakt"
	Priority = 900
)

const (
	baseURL    = "https://trakt.tv"
	apiBaseURL = "https://api.trakt.tv"

	moviePageURL  = "https://trakt.tv/movies/%s"
	personPageURL = "https://trakt.tv/people/%s"
)

type Trakt struct {
	*scraper.Scraper
	clientID string
}

func New() *Trakt {
	return &Trakt{
		Scraper: scraper.NewDefaultScraper(
			Name, baseURL, Priority, language.English,
		),
	}
}

func (t *Trakt) SetConfig(config provider.Config) error {
	if clientID, err := config.GetString("client_id"); err == nil {
		t.clientID = clientID
	}
	return nil
}

func (t *Trakt) resolveClientID() string {
	if k := t.GetRequestConfig().APIKeyFor(Name); k != "" {
		return k
	}
	return t.clientID
}

func (t *Trakt) apiGet(apiURL string, dest any) error {
	clientID := t.resolveClientID()
	if clientID == "" {
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
	headers.Set("trakt-api-key", clientID)
	headers.Set("trakt-api-version", "2")
	headers.Set("Content-Type", "application/json")

	if err := c.Request(http.MethodGet, apiURL, nil, nil, headers); err != nil {
		return err
	}
	return parseErr
}

func parseDate(s string) datatypes.Date {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return datatypes.Date(t)
	}
	return datatypes.Date(time.Time{})
}

func init() {
	provider.Register(Name, New)
}
