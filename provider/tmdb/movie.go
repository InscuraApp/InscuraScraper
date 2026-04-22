package tmdb

import (
	"fmt"
	"inscurascraper/model"
	"inscurascraper/provider"
	"net/url"
	"strconv"
)

// TMDB API response types for movies.

type searchMovieResponse struct {
	Page         int           `json:"page"`
	Results      []movieResult `json:"results"`
	TotalResults int           `json:"total_results"`
	TotalPages   int           `json:"total_pages"`
}

type movieResult struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	ReleaseDate   string  `json:"release_date"`
	VoteAverage   float64 `json:"vote_average"`
}

type movieDetailResponse struct {
	ID                  int                 `json:"id"`
	Title               string              `json:"title"`
	OriginalTitle       string              `json:"original_title"`
	Overview            string              `json:"overview"`
	PosterPath          string              `json:"poster_path"`
	BackdropPath        string              `json:"backdrop_path"`
	ReleaseDate         string              `json:"release_date"`
	Runtime             int                 `json:"runtime"`
	VoteAverage         float64             `json:"vote_average"`
	Genres              []genre             `json:"genres"`
	ProductionCompanies []productionCompany `json:"production_companies"`
	BelongsToCollection *collection         `json:"belongs_to_collection"`
	Credits             credits             `json:"credits"`
	Videos              videos              `json:"videos"`
}

type genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type productionCompany struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type collection struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type credits struct {
	Cast []castMember `json:"cast"`
	Crew []crewMember `json:"crew"`
}

type castMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	ProfilePath string `json:"profile_path"`
}

type crewMember struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Job        string `json:"job"`
	Department string `json:"department"`
}

type videos struct {
	Results []videoResult `json:"results"`
}

type videoResult struct {
	Key  string `json:"key"`
	Site string `json:"site"`
	Type string `json:"type"`
}

// NormalizeMovieID implements provider.MovieProvider.
func (t *TMDB) NormalizeMovieID(id string) string {
	if _, err := strconv.Atoi(id); err == nil {
		return id
	}
	return ""
}

// ParseMovieIDFromURL implements provider.MovieProvider.
func (t *TMDB) ParseMovieIDFromURL(rawURL string) (string, error) {
	return parseIDFromPath(rawURL)
}

// GetMovieInfoByID implements provider.MovieProvider.
func (t *TMDB) GetMovieInfoByID(id string) (*model.MovieInfo, error) {
	apiURL := fmt.Sprintf("%s/movie/%s?language=%s&append_to_response=credits,videos",
		apiBaseURL, id, t.resolveLanguage())

	resp := &movieDetailResponse{}
	if err := t.apiGet(apiURL, resp); err != nil {
		return nil, err
	}

	sid := strconv.Itoa(resp.ID)

	info := &model.MovieInfo{
		ID:            sid,
		Number:        sid,
		Provider:      t.Name(),
		Homepage:      fmt.Sprintf(moviePageURL, sid),
		Actors:        []string{},
		PreviewImages: []string{},
		Genres:        []string{},
	}

	info.Title = resp.Title
	if info.Title == "" {
		info.Title = resp.OriginalTitle
	}
	info.Summary = resp.Overview
	info.Score = resp.VoteAverage
	info.Runtime = resp.Runtime
	info.ReleaseDate = parseDate(resp.ReleaseDate)

	// Poster → Thumb
	info.ThumbURL = imageURL(posterW342, resp.PosterPath)
	info.BigThumbURL = imageURL(posterW500, resp.PosterPath)

	// Backdrop → Cover (fall back to poster if no backdrop)
	if resp.BackdropPath != "" {
		info.CoverURL = imageURL(backdropW780, resp.BackdropPath)
		info.BigCoverURL = imageURL(backdropOriginal, resp.BackdropPath)
	} else {
		info.CoverURL = imageURL(posterOriginal, resp.PosterPath)
		info.BigCoverURL = info.CoverURL
	}

	// Director from crew
	for _, crew := range resp.Credits.Crew {
		if crew.Job == "Director" {
			info.Director = crew.Name
			break
		}
	}

	// Cast → Actors (limit to 15)
	for i, cast := range resp.Credits.Cast {
		if i >= 15 {
			break
		}
		info.Actors = append(info.Actors, cast.Name)
	}

	// Genres
	for _, g := range resp.Genres {
		info.Genres = append(info.Genres, g.Name)
	}

	// Maker
	if len(resp.ProductionCompanies) > 0 {
		info.Maker = resp.ProductionCompanies[0].Name
	}

	// Series
	if resp.BelongsToCollection != nil {
		info.Series = resp.BelongsToCollection.Name
	}

	// Preview video (YouTube trailer)
	for _, v := range resp.Videos.Results {
		if v.Site == "YouTube" && v.Type == "Trailer" {
			info.PreviewVideoURL = "https://www.youtube.com/watch?v=" + v.Key
			break
		}
	}

	return info, nil
}

// GetMovieInfoByURL implements provider.MovieProvider.
func (t *TMDB) GetMovieInfoByURL(rawURL string) (*model.MovieInfo, error) {
	id, err := t.ParseMovieIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return t.GetMovieInfoByID(id)
}

// NormalizeMovieKeyword implements provider.MovieSearcher.
// The engine already strips release-name artefacts before calling us, so
// there's nothing provider-specific to normalise here.
func (t *TMDB) NormalizeMovieKeyword(keyword string) string {
	return keyword
}

// SearchMovie implements provider.MovieSearcher.
func (t *TMDB) SearchMovie(keyword string) ([]*model.MovieSearchResult, error) {
	apiURL := fmt.Sprintf("%s/search/movie?language=%s&query=%s",
		apiBaseURL, t.resolveLanguage(), url.QueryEscape(keyword))
	// Narrow results by year when the engine's release-name parser extracted one.
	if y := t.GetRequestConfig().SearchYearOr(0); y > 0 {
		apiURL += fmt.Sprintf("&year=%d", y)
	}

	resp := &searchMovieResponse{}
	if err := t.apiGet(apiURL, resp); err != nil {
		return nil, err
	}

	var results []*model.MovieSearchResult
	for _, r := range resp.Results {
		sid := strconv.Itoa(r.ID)

		coverURL := imageURL(backdropW780, r.BackdropPath)
		if coverURL == "" {
			coverURL = imageURL(posterOriginal, r.PosterPath)
		}

		results = append(results, &model.MovieSearchResult{
			ID:          sid,
			Number:      sid,
			Title:       r.Title,
			Provider:    t.Name(),
			Homepage:    fmt.Sprintf(moviePageURL, sid),
			ThumbURL:    imageURL(posterW342, r.PosterPath),
			CoverURL:    coverURL,
			Score:       r.VoteAverage,
			ReleaseDate: parseDate(r.ReleaseDate),
		})
	}

	if results == nil {
		return nil, provider.ErrInfoNotFound
	}

	return results, nil
}
