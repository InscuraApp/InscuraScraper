package trakt

import (
	"fmt"
	"inscurascraper/model"
	"inscurascraper/provider"
	"net/url"
	"path"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Trakt API response types for movies.

type traktIDs struct {
	Trakt int    `json:"trakt"`
	Slug  string `json:"slug"`
	IMDB  string `json:"imdb"`
	TMDB  int    `json:"tmdb"`
}

type searchMovieItem struct {
	Type  string      `json:"type"`
	Score float64     `json:"score"`
	Movie movieResult `json:"movie"`
}

type movieResult struct {
	Title    string   `json:"title"`
	Year     int      `json:"year"`
	IDs      traktIDs `json:"ids"`
	Overview string   `json:"overview"`
	Rating   float64  `json:"rating"`
	Runtime  int      `json:"runtime"`
	Released string   `json:"released"`
	Genres   []string `json:"genres"`
}

type movieDetailResponse struct {
	Title    string   `json:"title"`
	Year     int      `json:"year"`
	IDs      traktIDs `json:"ids"`
	Overview string   `json:"overview"`
	Rating   float64  `json:"rating"`
	Runtime  int      `json:"runtime"`
	Released string   `json:"released"`
	Genres   []string `json:"genres"`
}

type moviePeopleResponse struct {
	Cast []castEntry `json:"cast"`
	Crew struct {
		Directing []crewEntry `json:"directing"`
	} `json:"crew"`
}

type castEntry struct {
	Characters []string   `json:"characters"`
	Person     personStub `json:"person"`
}

type crewEntry struct {
	Jobs   []string   `json:"jobs"`
	Person personStub `json:"person"`
}

type personStub struct {
	Name string   `json:"name"`
	IDs  traktIDs `json:"ids"`
}

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// NormalizeMovieID implements provider.MovieProvider.
func (t *Trakt) NormalizeMovieID(id string) string {
	lower := strings.ToLower(id)
	if slugRe.MatchString(lower) {
		return lower
	}
	return ""
}

// ParseMovieIDFromURL implements provider.MovieProvider.
func (t *Trakt) ParseMovieIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", provider.ErrInvalidURL
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) < 2 || segments[0] != "movies" {
		return "", provider.ErrInvalidURL
	}
	slug := path.Base(u.Path)
	if t.NormalizeMovieID(slug) == "" {
		return "", provider.ErrInvalidURL
	}
	return slug, nil
}

// GetMovieInfoByID implements provider.MovieProvider.
func (t *Trakt) GetMovieInfoByID(id string) (*model.MovieInfo, error) {
	detail := &movieDetailResponse{}
	if err := t.apiGet(fmt.Sprintf("%s/movies/%s?extended=full", apiBaseURL, id), detail); err != nil {
		return nil, err
	}
	if detail.IDs.Slug == "" && detail.Title == "" {
		return nil, provider.ErrInfoNotFound
	}

	slug := detail.IDs.Slug
	if slug == "" {
		slug = id
	}

	info := &model.MovieInfo{
		ID:            slug,
		Number:        slug,
		Provider:      t.Name(),
		Homepage:      fmt.Sprintf(moviePageURL, slug),
		Title:         detail.Title,
		Summary:       detail.Overview,
		Score:         detail.Rating,
		Runtime:       detail.Runtime,
		ReleaseDate:   parseDate(detail.Released),
		Actors:        []string{},
		PreviewImages: []string{},
		Genres:        []string{},
	}

	for _, g := range detail.Genres {
		if g != "" {
			info.Genres = append(info.Genres, cases.Title(language.English).String(g))
		}
	}

	// Fetch cast and crew separately.
	people := &moviePeopleResponse{}
	if err := t.apiGet(fmt.Sprintf("%s/movies/%s/people", apiBaseURL, slug), people); err == nil {
		for _, c := range people.Crew.Directing {
			for _, job := range c.Jobs {
				if strings.EqualFold(job, "Director") {
					info.Director = c.Person.Name
					break
				}
			}
			if info.Director != "" {
				break
			}
		}
		for i, c := range people.Cast {
			if i >= 15 {
				break
			}
			info.Actors = append(info.Actors, c.Person.Name)
		}
	}

	return info, nil
}

// GetMovieInfoByURL implements provider.MovieProvider.
func (t *Trakt) GetMovieInfoByURL(rawURL string) (*model.MovieInfo, error) {
	id, err := t.ParseMovieIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return t.GetMovieInfoByID(id)
}

// NormalizeMovieKeyword implements provider.MovieSearcher.
func (t *Trakt) NormalizeMovieKeyword(keyword string) string {
	return keyword
}

// SearchMovie implements provider.MovieSearcher.
func (t *Trakt) SearchMovie(keyword string) ([]*model.MovieSearchResult, error) {
	apiURL := fmt.Sprintf("%s/search/movie?query=%s&extended=full", apiBaseURL, url.QueryEscape(keyword))

	var items []searchMovieItem
	if err := t.apiGet(apiURL, &items); err != nil {
		return nil, err
	}

	var results []*model.MovieSearchResult
	for _, item := range items {
		m := item.Movie
		slug := m.IDs.Slug
		if slug == "" {
			continue
		}
		results = append(results, &model.MovieSearchResult{
			ID:          slug,
			Number:      slug,
			Title:       m.Title,
			Provider:    t.Name(),
			Homepage:    fmt.Sprintf(moviePageURL, slug),
			Score:       m.Rating,
			ReleaseDate: parseDate(m.Released),
		})
	}

	if results == nil {
		return nil, provider.ErrInfoNotFound
	}
	return results, nil
}
