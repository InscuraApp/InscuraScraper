package tvdb

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"inscurascraper/model"
	"inscurascraper/provider"
)

// TVDB API response types for series and movies.

type seriesExtended struct {
	ID                   int           `json:"id"`
	Name                 string        `json:"name"`
	Slug                 string        `json:"slug"`
	Image                string        `json:"image"`
	FirstAired           string        `json:"firstAired"`
	LastAired            string        `json:"lastAired"`
	Year                 string        `json:"year"`
	Status               *status       `json:"status"`
	Score                int           `json:"score"`
	Runtime              int           `json:"averageRuntime"`
	OriginalCountry      string        `json:"originalCountry"`
	OriginalLanguage     string        `json:"originalLanguage"`
	Genres               []genre       `json:"genres"`
	Characters           []character   `json:"characters"`
	Artworks             []artwork     `json:"artworks"`
	Trailers             []trailer     `json:"trailers"`
	Companies            []company     `json:"companies"`
	RemoteIDs            []remoteID    `json:"remoteIds"`
	NameTranslations     []string      `json:"nameTranslations"`
	OverviewTranslations []string      `json:"overviewTranslations"`
	Translations         *translations `json:"translations"`
}

type movieExtended struct {
	ID                   int              `json:"id"`
	Name                 string           `json:"name"`
	Slug                 string           `json:"slug"`
	Image                string           `json:"image"`
	FirstRelease         *release         `json:"first_release"`
	Year                 string           `json:"year"`
	Status               *status          `json:"status"`
	Score                int              `json:"score"`
	Runtime              int              `json:"runtime"`
	OriginalCountry      string           `json:"originalCountry"`
	OriginalLanguage     string           `json:"originalLanguage"`
	Genres               []genre          `json:"genres"`
	Characters           []character      `json:"characters"`
	Artworks             []artwork        `json:"artworks"`
	Trailers             []trailer        `json:"trailers"`
	Studios              []company        `json:"studios"`
	Companies            *movieCompanies  `json:"companies"`
	RemoteIDs            []remoteID       `json:"remoteIds"`
	NameTranslations     []string         `json:"nameTranslations"`
	OverviewTranslations []string         `json:"overviewTranslations"`
	Translations         *translations    `json:"translations"`
}

// movieCompanies handles the nested company structure for movies:
// {"studio": [...], "production": [...], "distributor": [...], ...}
type movieCompanies struct {
	Studio       []company `json:"studio"`
	Production   []company `json:"production"`
	Distributor  []company `json:"distributor"`
	SpecialEffects []company `json:"special_effects"`
}

type translations struct {
	NameTranslations     []translation `json:"nameTranslations"`
	OverviewTranslations []translation `json:"overviewTranslations"`
}

type status struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type release struct {
	Country string `json:"country"`
	Date    string `json:"date"`
	Detail  string `json:"detail"`
}

type seriesBase struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type movieBase struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type searchResult struct {
	ObjectID        string   `json:"objectID"`
	Name            string   `json:"name"`
	Slug            string   `json:"slug"`
	Type            string   `json:"type"`
	TvdbID          string   `json:"tvdb_id"`
	Year            string   `json:"year"`
	ImageURL        string   `json:"image_url"`
	Thumbnail       string   `json:"thumbnail"`
	PrimaryLanguage string   `json:"primary_language"`
	Overview        string   `json:"overview"`
	Overviews       map[string]string `json:"overviews"`
	Translations    map[string]string `json:"translations"`
}

// Artwork type constants (from TVDB artwork types).
const (
	artworkTypeBanner    = 1
	artworkTypePoster    = 2
	artworkTypeBackground = 3
	artworkTypeIcon      = 5
)

// NormalizeMovieID implements provider.MovieProvider.
func (t *TVDB) NormalizeMovieID(id string) string {
	kind, numID := parseMovieID(id)
	if _, err := strconv.Atoi(numID); err != nil {
		return ""
	}
	return kind + ":" + numID
}

// ParseMovieIDFromURL implements provider.MovieProvider.
func (t *TVDB) ParseMovieIDFromURL(rawURL string) (string, error) {
	kind, slug, err := parseSlugFromURL(rawURL)
	if err != nil {
		return "", err
	}

	switch kind {
	case "series":
		var resp apiResponse[seriesBase]
		apiURL := fmt.Sprintf("%s/series/slug/%s", apiBaseURL, slug)
		if err := t.apiGet(apiURL, &resp); err != nil {
			return "", err
		}
		if resp.Data.ID == 0 {
			return "", provider.ErrInfoNotFound
		}
		return seriesPrefix + strconv.Itoa(resp.Data.ID), nil

	case "movies":
		var resp apiResponse[movieBase]
		apiURL := fmt.Sprintf("%s/movies/slug/%s", apiBaseURL, slug)
		if err := t.apiGet(apiURL, &resp); err != nil {
			return "", err
		}
		if resp.Data.ID == 0 {
			return "", provider.ErrInfoNotFound
		}
		return moviePrefix + strconv.Itoa(resp.Data.ID), nil

	default:
		return "", fmt.Errorf("unsupported TVDB URL type: %s", kind)
	}
}

// GetMovieInfoByID implements provider.MovieProvider.
func (t *TVDB) GetMovieInfoByID(id string) (*model.MovieInfo, error) {

	kind, numID := parseMovieID(id)

	switch kind {
	case "series":
		return t.getSeriesInfo(numID)
	case "movie":
		return t.getMovieInfo(numID)
	default:
		return nil, provider.ErrInvalidID
	}
}

func (t *TVDB) getSeriesInfo(id string) (*model.MovieInfo, error) {
	apiURL := fmt.Sprintf("%s/series/%s/extended", apiBaseURL, id)

	var resp apiResponse[seriesExtended]
	if err := t.apiGet(apiURL, &resp); err != nil {
		return nil, err
	}

	s := resp.Data
	if s.ID == 0 {
		return nil, provider.ErrInfoNotFound
	}

	sid := strconv.Itoa(s.ID)
	prefixedID := seriesPrefix + sid

	info := &model.MovieInfo{
		ID:            prefixedID,
		Number:        sid,
		Provider:      t.Name(),
		Homepage:      fmt.Sprintf(seriesPageURL, s.Slug),
		Actors:        []string{},
		PreviewImages: []string{},
		Genres:        []string{},
	}

	// Title: prefer resolved language translation, fallback to original name.
	info.Title = s.Name
	lang := t.resolveLanguage()
	if s.Translations != nil {
		if trName, trOverview := findTranslation(s.Translations.NameTranslations, lang); trName != "" {
			info.Title = trName
			_ = trOverview
		}
		if _, trOverview := findTranslation(s.Translations.OverviewTranslations, lang); trOverview != "" {
			info.Summary = trOverview
		}
	}
	if info.Summary == "" {
		// Fallback to English overview.
		if s.Translations != nil {
			if _, trOverview := findTranslation(s.Translations.OverviewTranslations, "eng"); trOverview != "" {
				info.Summary = trOverview
			}
		}
	}

	info.Score = float64(s.Score)
	info.Runtime = s.Runtime
	info.ReleaseDate = parseDate(s.FirstAired)

	// Primary image as cover.
	if s.Image != "" {
		info.CoverURL = s.Image
		info.BigCoverURL = s.Image
	}

	// Artworks: find poster for thumb, background for cover.
	for _, art := range s.Artworks {
		switch art.Type {
		case artworkTypePoster:
			if info.ThumbURL == "" {
				info.ThumbURL = art.Thumbnail
				info.BigThumbURL = art.Image
			}
			if info.CoverURL == "" {
				info.CoverURL = art.Image
				info.BigCoverURL = art.Image
			}
		case artworkTypeBackground:
			// Background overrides primary image as cover.
			info.CoverURL = art.Thumbnail
			info.BigCoverURL = art.Image
		}
		// Collect preview images (up to 10).
		if art.Type == artworkTypeBackground && len(info.PreviewImages) < 10 {
			info.PreviewImages = append(info.PreviewImages, art.Image)
		}
	}

	// If no thumb found, use cover.
	if info.ThumbURL == "" && info.CoverURL != "" {
		info.ThumbURL = info.CoverURL
		info.BigThumbURL = info.BigCoverURL
	}

	// Characters → Actors (limit 15).
	for i, ch := range s.Characters {
		if i >= 15 {
			break
		}
		if ch.PersonName != "" {
			info.Actors = append(info.Actors, ch.PersonName)
		}
	}

	// Genres.
	for _, g := range s.Genres {
		info.Genres = append(info.Genres, g.Name)
	}

	// Maker: first production company.
	for _, c := range s.Companies {
		info.Maker = c.Name
		break
	}

	// Trailer.
	for _, tr := range s.Trailers {
		if tr.URL != "" {
			info.PreviewVideoURL = tr.URL
			break
		}
	}

	return info, nil
}

func (t *TVDB) getMovieInfo(id string) (*model.MovieInfo, error) {
	apiURL := fmt.Sprintf("%s/movies/%s/extended", apiBaseURL, id)

	var resp apiResponse[movieExtended]
	if err := t.apiGet(apiURL, &resp); err != nil {
		return nil, err
	}

	m := resp.Data
	if m.ID == 0 {
		return nil, provider.ErrInfoNotFound
	}

	sid := strconv.Itoa(m.ID)
	prefixedID := moviePrefix + sid

	info := &model.MovieInfo{
		ID:            prefixedID,
		Number:        sid,
		Provider:      t.Name(),
		Homepage:      fmt.Sprintf(moviePageURL, m.Slug),
		Actors:        []string{},
		PreviewImages: []string{},
		Genres:        []string{},
	}

	// Title.
	info.Title = m.Name
	lang := t.resolveLanguage()
	if m.Translations != nil {
		if trName, _ := findTranslation(m.Translations.NameTranslations, lang); trName != "" {
			info.Title = trName
		}
		if _, trOverview := findTranslation(m.Translations.OverviewTranslations, lang); trOverview != "" {
			info.Summary = trOverview
		}
	}
	if info.Summary == "" && m.Translations != nil {
		if _, trOverview := findTranslation(m.Translations.OverviewTranslations, "eng"); trOverview != "" {
			info.Summary = trOverview
		}
	}

	info.Score = float64(m.Score)
	info.Runtime = m.Runtime

	if m.FirstRelease != nil {
		info.ReleaseDate = parseDate(m.FirstRelease.Date)
	}

	// Primary image as cover.
	if m.Image != "" {
		info.CoverURL = m.Image
		info.BigCoverURL = m.Image
	}

	// Artworks.
	for _, art := range m.Artworks {
		switch art.Type {
		case artworkTypePoster:
			if info.ThumbURL == "" {
				info.ThumbURL = art.Thumbnail
				info.BigThumbURL = art.Image
			}
		case artworkTypeBackground:
			info.CoverURL = art.Thumbnail
			info.BigCoverURL = art.Image
		}
		if art.Type == artworkTypeBackground && len(info.PreviewImages) < 10 {
			info.PreviewImages = append(info.PreviewImages, art.Image)
		}
	}

	if info.ThumbURL == "" && info.CoverURL != "" {
		info.ThumbURL = info.CoverURL
		info.BigThumbURL = info.BigCoverURL
	}

	// Director: character type 1 is usually Director in TVDB.
	for _, ch := range m.Characters {
		if ch.PersonName != "" && ch.Type == 1 {
			info.Director = ch.PersonName
			break
		}
	}

	// Actors (limit 15).
	for i, ch := range m.Characters {
		if i >= 15 {
			break
		}
		if ch.PersonName != "" {
			info.Actors = append(info.Actors, ch.PersonName)
		}
	}

	// Genres.
	for _, g := range m.Genres {
		info.Genres = append(info.Genres, g.Name)
	}

	// Maker: first studio or production company.
	if len(m.Studios) > 0 {
		info.Maker = m.Studios[0].Name
	} else if m.Companies != nil {
		if len(m.Companies.Studio) > 0 {
			info.Maker = m.Companies.Studio[0].Name
		} else if len(m.Companies.Production) > 0 {
			info.Maker = m.Companies.Production[0].Name
		}
	}

	// Trailer.
	for _, tr := range m.Trailers {
		if tr.URL != "" {
			info.PreviewVideoURL = tr.URL
			break
		}
	}

	return info, nil
}

// GetMovieInfoByURL implements provider.MovieProvider.
func (t *TVDB) GetMovieInfoByURL(rawURL string) (*model.MovieInfo, error) {
	id, err := t.ParseMovieIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return t.GetMovieInfoByID(id)
}

// NormalizeMovieKeyword implements provider.MovieSearcher.
// The engine already strips release-name artefacts before calling us.
func (t *TVDB) NormalizeMovieKeyword(keyword string) string {
	return keyword
}

// SearchMovie implements provider.MovieSearcher.
func (t *TVDB) SearchMovie(keyword string) ([]*model.MovieSearchResult, error) {

	apiURL := fmt.Sprintf("%s/search?query=%s", apiBaseURL, url.QueryEscape(keyword))

	var resp apiResponse[[]searchResult]
	if err := t.apiGet(apiURL, &resp); err != nil {
		return nil, err
	}

	var results []*model.MovieSearchResult
	for _, r := range resp.Data {
		if r.Type != "series" && r.Type != "movie" {
			continue
		}

		var prefix, pageURL string
		if r.Type == "series" {
			prefix = seriesPrefix
			pageURL = fmt.Sprintf(seriesPageURL, r.Slug)
		} else {
			prefix = moviePrefix
			pageURL = fmt.Sprintf(moviePageURL, r.Slug)
		}

		// Prefer resolved language title.
		title := r.Name
		lang := t.resolveLanguage()
		if trTitle, ok := r.Translations[lang]; ok && trTitle != "" {
			title = trTitle
		}

		// Prefer resolved language overview for display.
		overview := r.Overview
		if trOverview, ok := r.Overviews[lang]; ok && trOverview != "" {
			_ = trOverview
			overview = trOverview
		}
		_ = overview

		thumbURL := r.ImageURL
		if thumbURL == "" {
			thumbURL = r.Thumbnail
		}

		year := r.Year
		releaseDate := ""
		if len(year) == 4 {
			releaseDate = year + "-01-01"
		}

		tvdbID := r.TvdbID
		if tvdbID == "" {
			tvdbID = strings.TrimPrefix(r.ObjectID, r.Type+"-")
		}

		results = append(results, &model.MovieSearchResult{
			ID:          prefix + tvdbID,
			Number:      tvdbID,
			Title:       title,
			Provider:    t.Name(),
			Homepage:    pageURL,
			ThumbURL:    thumbURL,
			CoverURL:    thumbURL,
			ReleaseDate: parseDate(releaseDate),
		})
	}

	if results == nil {
		return nil, provider.ErrInfoNotFound
	}

	return results, nil
}
