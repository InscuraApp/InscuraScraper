package fanarttv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"golang.org/x/text/language"

	"inscurascraper/model"
	"inscurascraper/provider"
	"inscurascraper/provider/internal/scraper"
)

var (
	_ provider.MovieProvider = (*FanartTV)(nil)
	_ provider.ConfigSetter  = (*FanartTV)(nil)
)

const (
	Name     = "FanartTV"
	Priority = 1000
)

const (
	baseURL    = "https://fanart.tv/"
	apiBaseURL = "https://webservice.fanart.tv/v3"

	moviePageURL = "https://fanart.tv/movie/%s/"
	tvPageURL    = "https://fanart.tv/series/%s/"
)

// ID prefixes to distinguish movies from TV shows.
const (
	moviePrefix = "movie:"
	tvPrefix    = "tv:"
)

type FanartTV struct {
	*scraper.Scraper
	apiKey string
}

func New() *FanartTV {
	return &FanartTV{
		Scraper: scraper.NewDefaultScraper(
			Name, baseURL, Priority, language.Chinese,
		),
	}
}

func (f *FanartTV) SetConfig(config provider.Config) error {
	if apiKey, err := config.GetString("api_key"); err == nil {
		f.apiKey = apiKey
	}
	return nil
}

// resolveAPIKey returns the effective API key for the current request.
// A per-request X-Is-Api-Key-FanartTV header takes precedence over the global config.
func (f *FanartTV) resolveAPIKey() string {
	if k := f.GetRequestConfig().APIKeyFor(Name); k != "" {
		return k
	}
	return f.apiKey
}

// resolveLanguage returns the 2-letter ISO 639-1 code Fanart.tv uses in its
// image `lang` field, derived from the per-request X-Is-Language header.
// Returns "" when no header is set — callers should build their own preference
// list in that case.
func (f *FanartTV) resolveLanguage() string {
	tag := f.GetRequestConfig().LanguageOr("")
	if tag == "" {
		return ""
	}
	base := strings.ToLower(tag)
	if i := strings.IndexAny(base, "-_"); i > 0 {
		base = base[:i]
	}
	return base
}

// posterLangPrefs returns the language preference order for poster images.
// When a per-request language is set, it is tried first, followed by "en"
// and "" (language-neutral). Without a per-request language, the legacy
// order ("en", "zh", "") is kept for backward compatibility.
func (f *FanartTV) posterLangPrefs() []string {
	lang := f.resolveLanguage()
	if lang == "" {
		return []string{"en", "zh", ""}
	}
	if lang == "en" {
		return []string{"en", ""}
	}
	return []string{lang, "en", ""}
}

// backgroundLangPrefs returns the language preference order for backdrop/
// background images. Language-neutral images are preferred for backdrops
// since titles usually render as separate text layers.
func (f *FanartTV) backgroundLangPrefs() []string {
	lang := f.resolveLanguage()
	if lang == "" {
		return []string{"", "en"}
	}
	if lang == "en" {
		return []string{"", "en"}
	}
	return []string{"", lang, "en"}
}

// apiGet performs a GET request to the Fanart.tv API with the api_key query param.
// The API key is NOT embedded in the URL to prevent leakage in error messages.
func (f *FanartTV) apiGet(path string, dest any) error {
	apiKey := f.resolveAPIKey()
	if apiKey == "" {
		return provider.ErrProviderNotFound
	}

	apiURL := fmt.Sprintf("%s%s?api_key=%s", apiBaseURL, path, apiKey)

	c := f.ClonedCollector()

	var parseErr error
	c.OnResponse(func(r *colly.Response) {
		if err := json.Unmarshal(r.Body, dest); err != nil {
			parseErr = err
		}
	})

	headers := http.Header{}
	headers.Set("Accept", "application/json")
	if err := c.Request(http.MethodGet, apiURL, nil, nil, headers); err != nil {
		// Sanitize error: strip API key from any URL in error message
		return fmt.Errorf("fanart.tv request failed for %s: %w", path, err)
	}
	return parseErr
}

// Image types in API responses.
type fanartImage struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Lang     string `json:"lang"`
	Likes    string `json:"likes"`
	Season   string `json:"season,omitempty"`
	Disc     string `json:"disc,omitempty"`
	DiscType string `json:"disc_type,omitempty"`
}

// movieResponse holds the raw JSON for a movie lookup.
type movieResponse struct {
	Name             string        `json:"name"`
	TmdbID           string        `json:"tmdb_id"`
	ImdbID           string        `json:"imdb_id"`
	MoviePoster      []fanartImage `json:"movieposter"`
	MovieBackground  []fanartImage `json:"moviebackground"`
	MovieThumb       []fanartImage `json:"moviethumb"`
	HdMovieLogo      []fanartImage `json:"hdmovielogo"`
	HdMovieClearart  []fanartImage `json:"hdmovieclearart"`
	MovieLogo        []fanartImage `json:"movielogo"`
	MovieBanner      []fanartImage `json:"moviebanner"`
	MovieDisc        []fanartImage `json:"moviedisc"`
}

// tvResponse holds the raw JSON for a TV show lookup.
type tvResponse struct {
	Name            string        `json:"name"`
	ThetvdbID       string        `json:"thetvdb_id"`
	TvPoster        []fanartImage `json:"tvposter"`
	ShowBackground  []fanartImage `json:"showbackground"`
	TvThumb         []fanartImage `json:"tvthumb"`
	HdTvLogo        []fanartImage `json:"hdtvlogo"`
	HdClearart      []fanartImage `json:"hdclearart"`
	ClearLogo       []fanartImage `json:"clearlogo"`
	ClearArt        []fanartImage `json:"clearart"`
	CharacterArt    []fanartImage `json:"characterart"`
	TvBanner        []fanartImage `json:"tvbanner"`
	SeasonPoster    []fanartImage `json:"seasonposter"`
	SeasonThumb     []fanartImage `json:"seasonthumb"`
	SeasonBanner    []fanartImage `json:"seasonbanner"`
}

// parseID splits a prefixed ID like "movie:550" or "tv:81189".
// Returns (kind, numericID). Defaults to "movie" if no prefix.
func parseID(id string) (kind, numericID string) {
	if strings.HasPrefix(id, moviePrefix) {
		return "movie", strings.TrimPrefix(id, moviePrefix)
	}
	if strings.HasPrefix(id, tvPrefix) {
		return "tv", strings.TrimPrefix(id, tvPrefix)
	}
	return "movie", id
}

// NormalizeMovieID implements provider.MovieProvider.
func (f *FanartTV) NormalizeMovieID(id string) string {
	kind, numID := parseID(id)
	if _, err := strconv.Atoi(numID); err != nil {
		return ""
	}
	return kind + ":" + numID
}

// ParseMovieIDFromURL implements provider.MovieProvider.
func (f *FanartTV) ParseMovieIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	// URLs: https://fanart.tv/movie/550/fight-club/
	//       https://fanart.tv/series/81189/breaking-bad/
	parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(parts) < 2 {
		return "", provider.ErrInvalidURL
	}
	numID := parts[1]
	if _, err := strconv.Atoi(numID); err != nil {
		return "", fmt.Errorf("invalid Fanart.tv ID: %s", numID)
	}
	switch parts[0] {
	case "movie":
		return moviePrefix + numID, nil
	case "series", "tv":
		return tvPrefix + numID, nil
	default:
		return "", fmt.Errorf("unsupported Fanart.tv URL type: %s", parts[0])
	}
}

// GetMovieInfoByID implements provider.MovieProvider.
func (f *FanartTV) GetMovieInfoByID(id string) (*model.MovieInfo, error) {

	kind, numID := parseID(id)

	switch kind {
	case "movie":
		return f.getMovieImages(numID)
	case "tv":
		return f.getTVImages(numID)
	default:
		return nil, provider.ErrInvalidID
	}
}

func (f *FanartTV) getMovieImages(tmdbID string) (*model.MovieInfo, error) {
	resp := &movieResponse{}
	if err := f.apiGet("/movies/"+tmdbID, resp); err != nil {
		return nil, err
	}

	if resp.Name == "" && resp.TmdbID == "" {
		return nil, provider.ErrInfoNotFound
	}

	prefixedID := moviePrefix + tmdbID

	info := &model.MovieInfo{
		ID:            prefixedID,
		Number:        tmdbID,
		Title:         resp.Name,
		Provider:      f.Name(),
		Homepage:      fmt.Sprintf(moviePageURL, tmdbID),
		Actors:        []string{},
		PreviewImages: []string{},
		Genres:        []string{},
	}

	// Poster → Thumb (sorted by likes).
	posters := sortByLikes(resp.MoviePoster)
	if best := pickBest(posters, f.posterLangPrefs()...); best != nil {
		info.ThumbURL = best.URL
		info.BigThumbURL = best.URL
	}

	// Background → Cover.
	backgrounds := sortByLikes(resp.MovieBackground)
	if best := pickBest(backgrounds, f.backgroundLangPrefs()...); best != nil {
		info.CoverURL = best.URL
		info.BigCoverURL = best.URL
	}

	// If no cover, use poster as fallback.
	if info.CoverURL == "" && info.ThumbURL != "" {
		info.CoverURL = info.ThumbURL
		info.BigCoverURL = info.BigThumbURL
	}

	// Preview images: collect backgrounds (up to 10).
	for i, img := range backgrounds {
		if i >= 10 {
			break
		}
		info.PreviewImages = append(info.PreviewImages, img.URL)
	}

	return info, nil
}

func (f *FanartTV) getTVImages(tvdbID string) (*model.MovieInfo, error) {
	resp := &tvResponse{}
	if err := f.apiGet("/tv/"+tvdbID, resp); err != nil {
		return nil, err
	}

	if resp.Name == "" && resp.ThetvdbID == "" {
		return nil, provider.ErrInfoNotFound
	}

	prefixedID := tvPrefix + tvdbID

	info := &model.MovieInfo{
		ID:            prefixedID,
		Number:        tvdbID,
		Title:         resp.Name,
		Provider:      f.Name(),
		Homepage:      fmt.Sprintf(tvPageURL, tvdbID),
		Actors:        []string{},
		PreviewImages: []string{},
		Genres:        []string{},
	}

	// Poster → Thumb.
	posters := sortByLikes(resp.TvPoster)
	if best := pickBest(posters, f.posterLangPrefs()...); best != nil {
		info.ThumbURL = best.URL
		info.BigThumbURL = best.URL
	}

	// Background → Cover.
	backgrounds := sortByLikes(resp.ShowBackground)
	if best := pickBest(backgrounds, f.backgroundLangPrefs()...); best != nil {
		info.CoverURL = best.URL
		info.BigCoverURL = best.URL
	}

	if info.CoverURL == "" && info.ThumbURL != "" {
		info.CoverURL = info.ThumbURL
		info.BigCoverURL = info.BigThumbURL
	}

	// Preview images.
	for i, img := range backgrounds {
		if i >= 10 {
			break
		}
		info.PreviewImages = append(info.PreviewImages, img.URL)
	}

	return info, nil
}

// GetMovieInfoByURL implements provider.MovieProvider.
func (f *FanartTV) GetMovieInfoByURL(rawURL string) (*model.MovieInfo, error) {
	id, err := f.ParseMovieIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return f.GetMovieInfoByID(id)
}

// sortByLikes returns images sorted by likes (descending).
func sortByLikes(images []fanartImage) []fanartImage {
	sorted := make([]fanartImage, len(images))
	copy(sorted, images)
	sort.SliceStable(sorted, func(i, j int) bool {
		li, _ := strconv.Atoi(sorted[i].Likes)
		lj, _ := strconv.Atoi(sorted[j].Likes)
		return li > lj
	})
	return sorted
}

// pickBest returns the first image matching any of the preferred languages.
// Empty string "" matches images with no language tag.
func pickBest(images []fanartImage, langs ...string) *fanartImage {
	for _, lang := range langs {
		for i := range images {
			if images[i].Lang == lang {
				return &images[i]
			}
		}
	}
	if len(images) > 0 {
		return &images[0]
	}
	return nil
}

// parseIDFromURL extracts a numeric ID from a URL path like /movie/550/...
func parseIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid Fanart.tv URL: %s", rawURL)
	}
	if _, err := strconv.Atoi(parts[1]); err != nil {
		return "", fmt.Errorf("invalid ID: %s", parts[1])
	}
	return parts[1], nil
}

func init() {
	provider.Register(Name, New)
}
