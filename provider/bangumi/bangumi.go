package bangumi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"inscurascraper/provider"
	"inscurascraper/provider/internal/scraper"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"golang.org/x/text/language"
	"gorm.io/datatypes"
)

const (
	Name     = "Bangumi"
	Priority = 900
)

const (
	baseURL = "https://bgm.tv/"
	apiBase = "https://api.bgm.tv"
)

// bgmImages represents the image set returned by Bangumi API.
type bgmImages struct {
	Large  string `json:"large"`
	Common string `json:"common"`
	Medium string `json:"medium"`
	Small  string `json:"small"`
	Grid   string `json:"grid"`
}

// infoboxItem represents a key-value pair in the Bangumi infobox.
// The value field can be either a JSON string or a JSON array.
type infoboxItem struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

// Bangumi is the provider for bgm.tv (番组计划).
type Bangumi struct {
	*scraper.Scraper
}

// New creates a new Bangumi provider.
func New() *Bangumi {
	return &Bangumi{
		Scraper: scraper.NewDefaultScraper(
			Name, baseURL, Priority, language.Chinese,
		),
	}
}

func init() {
	provider.Register(Name, New)
}

// defaultHeaders returns the common HTTP headers required by the Bangumi API.
func defaultHeaders() http.Header {
	h := http.Header{}
	h.Set("Accept", "application/json")
	h.Set("User-Agent", "inscurascraper/1.0")
	return h
}

// apiGet performs a GET request to the Bangumi API and unmarshals the response into dest.
func (b *Bangumi) apiGet(apiURL string, dest any) error {
	c := b.ClonedCollector()
	var parseErr error
	c.OnResponse(func(r *colly.Response) {
		if err := json.Unmarshal(r.Body, dest); err != nil {
			parseErr = err
		}
	})
	if err := c.Request(http.MethodGet, apiURL, nil, nil, defaultHeaders()); err != nil {
		return err
	}
	return parseErr
}

// apiPost performs a POST request with a JSON body and unmarshals the response into dest.
func (b *Bangumi) apiPost(apiURL string, reqBody, dest any) error {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("bangumi: marshal request body: %w", err)
	}
	c := b.ClonedCollector()
	var parseErr error
	c.OnResponse(func(r *colly.Response) {
		if err := json.Unmarshal(r.Body, dest); err != nil {
			parseErr = err
		}
	})
	h := defaultHeaders()
	h.Set("Content-Type", "application/json")
	if err := c.Request(http.MethodPost, apiURL, bytes.NewReader(bodyBytes), nil, h); err != nil {
		return err
	}
	return parseErr
}

// parseDate parses a "2006-01-02" date string into a datatypes.Date.
func parseDate(s string) datatypes.Date {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return datatypes.Date(t)
	}
	return datatypes.Date(time.Time{})
}

// parseIDFromURL extracts a numeric ID from a Bangumi URL such as
// https://bgm.tv/subject/{id} or https://bgm.tv/person/{id}.
// segment must be "subject" or "person".
func parseIDFromURL(rawURL, segment string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", provider.ErrInvalidURL
	}
	parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(parts) < 2 || parts[0] != segment {
		return "", provider.ErrInvalidURL
	}
	if _, err := strconv.Atoi(parts[1]); err != nil {
		return "", provider.ErrInvalidURL
	}
	return parts[1], nil
}

// infoboxGet returns the first value for key in items.
// The infobox value may be a plain string or an array of {"v": "..."} objects.
func infoboxGet(items []infoboxItem, key string) string {
	for _, item := range items {
		if item.Key != key {
			continue
		}
		var s string
		if err := json.Unmarshal(item.Value, &s); err == nil {
			return s
		}
		var arr []map[string]string
		if err := json.Unmarshal(item.Value, &arr); err == nil && len(arr) > 0 {
			if v, ok := arr[0]["v"]; ok {
				return v
			}
		}
	}
	return ""
}

// infoboxGetAll returns all values for key in items.
func infoboxGetAll(items []infoboxItem, key string) []string {
	for _, item := range items {
		if item.Key != key {
			continue
		}
		var s string
		if err := json.Unmarshal(item.Value, &s); err == nil {
			return []string{s}
		}
		var arr []map[string]string
		if err := json.Unmarshal(item.Value, &arr); err == nil {
			var vals []string
			for _, m := range arr {
				if v := m["v"]; v != "" {
					vals = append(vals, v)
				}
			}
			return vals
		}
	}
	return nil
}

// extractImageURLs returns a deduplicated list of image URLs from a bgmImages value.
func extractImageURLs(imgs bgmImages) []string {
	var out []string
	seen := make(map[string]bool)
	for _, u := range []string{imgs.Large, imgs.Medium} {
		if u != "" && !seen[u] {
			out = append(out, u)
			seen[u] = true
		}
	}
	return out
}

// bloodTypeStr converts the Bangumi integer blood type to its letter representation.
func bloodTypeStr(t int) string {
	switch t {
	case 1:
		return "A"
	case 2:
		return "B"
	case 3:
		return "AB"
	case 4:
		return "O"
	default:
		return ""
	}
}
