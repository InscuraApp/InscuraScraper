package anilist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
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
	_ provider.MovieProvider = (*AniList)(nil)
	_ provider.MovieSearcher = (*AniList)(nil)
	_ provider.ActorProvider = (*AniList)(nil)
	_ provider.ActorSearcher = (*AniList)(nil)
)

const (
	Name     = "AniList"
	Priority = 1000
)

const (
	baseURL       = "https://anilist.co/"
	graphqlURL    = "https://graphql.anilist.co"
	animePageURL  = "https://anilist.co/anime/%s"
	mangaPageURL  = "https://anilist.co/manga/%s"
	staffPageURL  = "https://anilist.co/staff/%s"
)

// ID prefixes to distinguish anime from manga.
const (
	animePrefix = "anime:"
	mangaPrefix = "manga:"
)

type AniList struct {
	*scraper.Scraper
}

func New() *AniList {
	return &AniList{
		Scraper: scraper.NewDefaultScraper(
			Name, baseURL, Priority, language.Japanese,
		),
	}
}

// bcp47ToAniListVoiceLang maps a BCP 47 tag to the AniList StaffLanguage
// enum accepted by Media.characters.voiceActors(language: …). AniList only
// supports a small fixed set; unknown tags fall back to JAPANESE which is
// what the anime/manga scene uses by convention.
var bcp47ToAniListVoiceLang = map[string]string{
	"ja": "JAPANESE",
	"en": "ENGLISH",
	"ko": "KOREAN",
	"it": "ITALIAN",
	"es": "SPANISH",
	"pt": "PORTUGUESE",
	"fr": "FRENCH",
	"de": "GERMAN",
	"he": "HEBREW",
	"hu": "HUNGARIAN",
}

// resolveVoiceActorLanguage returns the AniList voice-actor enum to use in
// the GraphQL query for this request. Falls back to JAPANESE when the
// X-Is-Language header is absent or maps to an unsupported language.
func (a *AniList) resolveVoiceActorLanguage() string {
	tag := a.GetRequestConfig().LanguageOr("")
	if tag == "" {
		return "JAPANESE"
	}
	base := strings.ToLower(tag)
	if i := strings.IndexAny(base, "-_"); i > 0 {
		base = base[:i]
	}
	if enum, ok := bcp47ToAniListVoiceLang[base]; ok {
		return enum
	}
	return "JAPANESE"
}

// resolveTitle picks a localized title from AniList's three-field title
// object based on the X-Is-Language header:
//   - "en*" → prefer English, then Romaji, then Native
//   - "ja*" → prefer Native (Japanese kanji), then Romaji, then English
//   - anything else (including no header) → keep the legacy English→Romaji→Native order
func (a *AniList) resolveTitle(t mediaTitle) string {
	tag := strings.ToLower(a.GetRequestConfig().LanguageOr(""))
	base := tag
	if i := strings.IndexAny(base, "-_"); i > 0 {
		base = base[:i]
	}
	switch base {
	case "ja":
		for _, s := range []string{t.Native, t.Romaji, t.English} {
			if s != "" {
				return s
			}
		}
	case "en", "":
		for _, s := range []string{t.English, t.Romaji, t.Native} {
			if s != "" {
				return s
			}
		}
	default:
		// Other languages: AniList has no localized title field — userPreferred
		// is only meaningful for authenticated users, so we fall back to
		// English → Romaji → Native.
		for _, s := range []string{t.English, t.Romaji, t.Native} {
			if s != "" {
				return s
			}
		}
	}
	return ""
}

// isJapanese returns true when the given BCP 47 tag starts with "ja".
func isJapanese(tag string) bool {
	tag = strings.ToLower(tag)
	return tag == "ja" || strings.HasPrefix(tag, "ja-") || strings.HasPrefix(tag, "ja_")
}

// graphqlQuery executes a GraphQL query against AniList API.
func (a *AniList) graphqlQuery(query string, variables map[string]any, dest any) error {
	payload, _ := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})

	c := a.ClonedCollector()

	var parseErr error
	c.OnResponse(func(r *colly.Response) {
		if err := json.Unmarshal(r.Body, dest); err != nil {
			parseErr = err
		}
	})

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", "application/json")
	if err := c.Request(http.MethodPost, graphqlURL, bytes.NewReader(payload), nil, headers); err != nil {
		return err
	}
	return parseErr
}

// parseDate converts AniList date object to datatypes.Date.
func parseDate(d *fuzzyDate) datatypes.Date {
	if d == nil || d.Year == nil {
		return datatypes.Date(time.Time{})
	}
	year := *d.Year
	month := 1
	day := 1
	if d.Month != nil {
		month = *d.Month
	}
	if d.Day != nil {
		day = *d.Day
	}
	return datatypes.Date(time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC))
}

// stripMarkdownLinks removes [text](url) patterns, keeping only text.
var markdownLinkRe = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)

// stripHTML removes HTML tags.
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func cleanDescription(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = markdownLinkRe.ReplaceAllString(s, "$1")
	return strings.TrimSpace(s)
}

// parseID splits "anime:123" or "manga:456". Default to "anime".
func parseID(id string) (kind, numericID string) {
	if strings.HasPrefix(id, animePrefix) {
		return "anime", strings.TrimPrefix(id, animePrefix)
	}
	if strings.HasPrefix(id, mangaPrefix) {
		return "manga", strings.TrimPrefix(id, mangaPrefix)
	}
	return "anime", id
}

// parseIDFromURL extracts type and ID from AniList URLs like
// https://anilist.co/anime/1/Cowboy-Bebop
func parseIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(parts) < 2 {
		return "", provider.ErrInvalidURL
	}
	kind := parts[0]
	numID := parts[1]
	if _, err := strconv.Atoi(numID); err != nil {
		return "", fmt.Errorf("invalid AniList ID: %s", numID)
	}
	switch kind {
	case "anime":
		return animePrefix + numID, nil
	case "manga":
		return mangaPrefix + numID, nil
	case "staff":
		return numID, nil
	default:
		return "", fmt.Errorf("unsupported AniList URL type: %s", kind)
	}
}

// parseStaffIDFromURL extracts staff ID from https://anilist.co/staff/97009/...
func parseStaffIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(parts) < 2 || parts[0] != "staff" {
		return "", provider.ErrInvalidURL
	}
	numID := parts[1]
	if _, err := strconv.Atoi(numID); err != nil {
		return "", fmt.Errorf("invalid AniList staff ID: %s", numID)
	}
	return numID, nil
}

// parseCharacterIDFromURL for https://anilist.co/character/1/...
func parseCharacterIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	base := path.Base(u.Path)
	if idx := strings.Index(base, "-"); idx > 0 {
		base = base[:idx]
	}
	if _, err := strconv.Atoi(base); err != nil {
		// Try second path segment
		parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
		if len(parts) >= 2 {
			if _, err := strconv.Atoi(parts[1]); err == nil {
				return parts[1], nil
			}
		}
		return "", fmt.Errorf("invalid AniList ID: %s", base)
	}
	return base, nil
}

// Common GraphQL response types.

type fuzzyDate struct {
	Year  *int `json:"year"`
	Month *int `json:"month"`
	Day   *int `json:"day"`
}

type mediaTitle struct {
	Romaji        string `json:"romaji"`
	English       string `json:"english"`
	Native        string `json:"native"`
	UserPreferred string `json:"userPreferred"`
}

type coverImage struct {
	ExtraLarge string `json:"extraLarge"`
	Large      string `json:"large"`
	Medium     string `json:"medium"`
}

type staffImage struct {
	Large  string `json:"large"`
	Medium string `json:"medium"`
}

type characterImage struct {
	Large  string `json:"large"`
	Medium string `json:"medium"`
}

type studioNode struct {
	Name              string `json:"name"`
	IsAnimationStudio bool   `json:"isAnimationStudio"`
}

type trailerObj struct {
	ID        string `json:"id"`
	Site      string `json:"site"`
	Thumbnail string `json:"thumbnail"`
}

type staffName struct {
	Full          string `json:"full"`
	Native        string `json:"native"`
	UserPreferred string `json:"userPreferred"`
}

func init() {
	provider.Register(Name, New)
}
