package tvmaze

import (
	"fmt"
	"net/url"
	"strconv"

	"inscurascraper/model"
	"inscurascraper/provider"
)

// TVMaze API response types for shows.

type showResponse struct {
	ID           int         `json:"id"`
	URL          string      `json:"url"`
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Language     string      `json:"language"`
	Genres       []string    `json:"genres"`
	Status       string      `json:"status"`
	Runtime      *int        `json:"runtime"`
	Premiered    string      `json:"premiered"`
	Ended        string      `json:"ended"`
	OfficialSite *string     `json:"officialSite"`
	Rating       rating      `json:"rating"`
	Weight       int         `json:"weight"`
	Network      *network    `json:"network"`
	WebChannel   *webChannel `json:"webChannel"`
	Externals    externals   `json:"externals"`
	Image        *imageObj   `json:"image"`
	Summary      string      `json:"summary"`
	Embedded     *embedded   `json:"_embedded"`
}

type embedded struct {
	Cast []castItem `json:"cast"`
}

type castItem struct {
	Person    personBrief   `json:"person"`
	Character characterBrief `json:"character"`
}

type personBrief struct {
	ID       int       `json:"id"`
	URL      string    `json:"url"`
	Name     string    `json:"name"`
	Country  *country  `json:"country"`
	Birthday string    `json:"birthday"`
	Deathday *string   `json:"deathday"`
	Gender   string    `json:"gender"`
	Image    *imageObj `json:"image"`
}

type characterBrief struct {
	ID    int       `json:"id"`
	URL   string    `json:"url"`
	Name  string    `json:"name"`
	Image *imageObj `json:"image"`
}

type searchShowResult struct {
	Score float64      `json:"score"`
	Show  showResponse `json:"show"`
}

// NormalizeMovieID implements provider.MovieProvider.
func (t *TVMaze) NormalizeMovieID(id string) string {
	if _, err := strconv.Atoi(id); err == nil {
		return id
	}
	return ""
}

// ParseMovieIDFromURL implements provider.MovieProvider.
func (t *TVMaze) ParseMovieIDFromURL(rawURL string) (string, error) {
	return parseIDFromURL(rawURL)
}

// GetMovieInfoByID implements provider.MovieProvider.
func (t *TVMaze) GetMovieInfoByID(id string) (*model.MovieInfo, error) {
	apiURL := fmt.Sprintf("%s/shows/%s?embed=cast", apiBaseURL, id)

	resp := &showResponse{}
	if err := t.apiGet(apiURL, resp); err != nil {
		return nil, err
	}

	if resp.ID == 0 {
		return nil, provider.ErrInfoNotFound
	}

	sid := strconv.Itoa(resp.ID)

	info := &model.MovieInfo{
		ID:            sid,
		Number:        sid,
		Title:         resp.Name,
		Provider:      t.Name(),
		Homepage:      resp.URL,
		Actors:        []string{},
		PreviewImages: []string{},
		Genres:        resp.Genres,
	}

	if info.Genres == nil {
		info.Genres = []string{}
	}

	info.Summary = stripHTML(resp.Summary)

	if resp.Rating.Average != nil {
		info.Score = *resp.Rating.Average
	}
	if resp.Runtime != nil {
		info.Runtime = *resp.Runtime
	}
	info.ReleaseDate = parseDate(resp.Premiered)

	// Image: poster as thumb.
	if resp.Image != nil {
		info.ThumbURL = resp.Image.Medium
		info.BigThumbURL = resp.Image.Original
		info.CoverURL = resp.Image.Original
		info.BigCoverURL = resp.Image.Original
	}

	// Network as maker.
	if resp.Network != nil {
		info.Maker = resp.Network.Name
	} else if resp.WebChannel != nil {
		info.Maker = resp.WebChannel.Name
	}

	// Cast → Actors (limit 15).
	if resp.Embedded != nil {
		for i, cast := range resp.Embedded.Cast {
			if i >= 15 {
				break
			}
			info.Actors = append(info.Actors, cast.Person.Name)
		}
	}

	// Fetch show images for backgrounds.
	t.enrichImages(sid, info)

	return info, nil
}

// enrichImages fetches /shows/{id}/images and fills in background images.
func (t *TVMaze) enrichImages(id string, info *model.MovieInfo) {
	apiURL := fmt.Sprintf("%s/shows/%s/images", apiBaseURL, id)

	var images []showImage
	if err := t.apiGet(apiURL, &images); err != nil {
		return
	}

	for _, img := range images {
		if img.Type == "background" && img.Resolutions.Original != nil {
			// Use first background as cover.
			if info.CoverURL == info.ThumbURL || info.CoverURL == info.BigThumbURL {
				info.CoverURL = img.Resolutions.Original.URL
				info.BigCoverURL = img.Resolutions.Original.URL
			}
			if len(info.PreviewImages) < 10 {
				info.PreviewImages = append(info.PreviewImages, img.Resolutions.Original.URL)
			}
		}
	}
}

// GetMovieInfoByURL implements provider.MovieProvider.
func (t *TVMaze) GetMovieInfoByURL(rawURL string) (*model.MovieInfo, error) {
	id, err := t.ParseMovieIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return t.GetMovieInfoByID(id)
}

// NormalizeMovieKeyword implements provider.MovieSearcher.
// The engine already strips release-name artefacts before calling us.
func (t *TVMaze) NormalizeMovieKeyword(keyword string) string {
	return keyword
}

// SearchMovie implements provider.MovieSearcher.
func (t *TVMaze) SearchMovie(keyword string) ([]*model.MovieSearchResult, error) {
	apiURL := fmt.Sprintf("%s/search/shows?q=%s", apiBaseURL, url.QueryEscape(keyword))

	var resp []searchShowResult
	if err := t.apiGet(apiURL, &resp); err != nil {
		return nil, err
	}

	var results []*model.MovieSearchResult
	for _, r := range resp {
		s := r.Show
		sid := strconv.Itoa(s.ID)

		var thumbURL, coverURL string
		if s.Image != nil {
			thumbURL = s.Image.Medium
			coverURL = s.Image.Original
		}

		var score float64
		if s.Rating.Average != nil {
			score = *s.Rating.Average
		}

		results = append(results, &model.MovieSearchResult{
			ID:          sid,
			Number:      sid,
			Title:       s.Name,
			Provider:    t.Name(),
			Homepage:    s.URL,
			ThumbURL:    thumbURL,
			CoverURL:    coverURL,
			Score:       score,
			ReleaseDate: parseDate(s.Premiered),
		})
	}

	if results == nil {
		return nil, provider.ErrInfoNotFound
	}

	return results, nil
}
