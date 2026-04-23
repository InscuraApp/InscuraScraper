package bangumi

import (
	"fmt"
	"inscurascraper/model"
	"inscurascraper/provider"
	"strconv"
	"strings"
)

var (
	_ provider.MovieProvider = (*Bangumi)(nil)
	_ provider.MovieSearcher = (*Bangumi)(nil)
)

const subjectTypeAnime = 2

type subjectRating struct {
	Rank  int     `json:"rank"`
	Total int     `json:"total"`
	Score float64 `json:"score"`
}

type subjectTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type subject struct {
	ID            int           `json:"id"`
	Name          string        `json:"name"`
	NameCN        string        `json:"name_cn"`
	Summary       string        `json:"summary"`
	Date          string        `json:"date"`
	Images        bgmImages     `json:"images"`
	Rating        subjectRating `json:"rating"`
	Tags          []subjectTag  `json:"tags"`
	TotalEpisodes int           `json:"total_episodes"`
	Infobox       []infoboxItem `json:"infobox"`
}

type subjectPerson struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	Relation string `json:"relation"`
}

type searchSubjectsFilter struct {
	Type []int `json:"type"`
}

type searchSubjectsRequest struct {
	Keyword string               `json:"keyword"`
	Filter  searchSubjectsFilter `json:"filter"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
}

type searchSubjectsResponse struct {
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
	Data   []subject `json:"data"`
}

func (b *Bangumi) NormalizeMovieID(id string) string {
	if _, err := strconv.Atoi(id); err != nil {
		return ""
	}
	return id
}

func (b *Bangumi) ParseMovieIDFromURL(rawURL string) (string, error) {
	return parseIDFromURL(rawURL, "subject")
}

func (b *Bangumi) GetMovieInfoByID(id string) (*model.MovieInfo, error) {
	if b.NormalizeMovieID(id) == "" {
		return nil, provider.ErrInvalidID
	}

	var s subject
	if err := b.apiGet(fmt.Sprintf("%s/v0/subjects/%s", apiBase, id), &s); err != nil {
		return nil, err
	}
	if s.ID == 0 {
		return nil, provider.ErrInfoNotFound
	}

	var persons []subjectPerson
	_ = b.apiGet(fmt.Sprintf("%s/v0/subjects/%s/persons", apiBase, id), &persons)

	return subjectToMovieInfo(&s, persons), nil
}

func (b *Bangumi) GetMovieInfoByURL(rawURL string) (*model.MovieInfo, error) {
	id, err := b.ParseMovieIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return b.GetMovieInfoByID(id)
}

func (b *Bangumi) NormalizeMovieKeyword(keyword string) string {
	return strings.TrimSpace(keyword)
}

func (b *Bangumi) SearchMovie(keyword string) ([]*model.MovieSearchResult, error) {
	keyword = b.NormalizeMovieKeyword(keyword)
	if keyword == "" {
		return nil, provider.ErrInvalidKeyword
	}

	req := searchSubjectsRequest{
		Keyword: keyword,
		Filter:  searchSubjectsFilter{Type: []int{subjectTypeAnime}},
		Limit:   25,
		Offset:  0,
	}
	var resp searchSubjectsResponse
	if err := b.apiPost(fmt.Sprintf("%s/v0/search/subjects", apiBase), req, &resp); err != nil {
		return nil, err
	}

	var results []*model.MovieSearchResult
	for i := range resp.Data {
		r := subjectToSearchResult(&resp.Data[i])
		if r.IsValid() {
			results = append(results, r)
		}
	}
	return results, nil
}

func subjectTitle(s *subject) string {
	if s.NameCN != "" {
		return s.NameCN
	}
	return s.Name
}

func subjectToSearchResult(s *subject) *model.MovieSearchResult {
	id := strconv.Itoa(s.ID)
	return &model.MovieSearchResult{
		ID:          id,
		Number:      id,
		Title:       subjectTitle(s),
		Provider:    Name,
		Homepage:    fmt.Sprintf("%ssubject/%d", baseURL, s.ID),
		ThumbURL:    s.Images.Medium,
		CoverURL:    s.Images.Common,
		Score:       s.Rating.Score,
		ReleaseDate: parseDate(s.Date),
	}
}

func subjectToMovieInfo(s *subject, persons []subjectPerson) *model.MovieInfo {
	id := strconv.Itoa(s.ID)

	var actors []string
	var director string
	for _, p := range persons {
		switch p.Relation {
		case "出演":
			actors = append(actors, p.Name)
		case "导演":
			if director == "" {
				director = p.Name
			}
		}
	}

	var genres []string
	for i, tag := range s.Tags {
		if i >= 10 {
			break
		}
		genres = append(genres, tag.Name)
	}

	return &model.MovieInfo{
		ID:          id,
		Number:      id,
		Title:       subjectTitle(s),
		Summary:     s.Summary,
		Provider:    Name,
		Homepage:    fmt.Sprintf("%ssubject/%d", baseURL, s.ID),
		ThumbURL:    s.Images.Medium,
		BigThumbURL: s.Images.Large,
		CoverURL:    s.Images.Common,
		BigCoverURL: s.Images.Large,
		Score:       s.Rating.Score,
		ReleaseDate: parseDate(s.Date),
		Director:    director,
		Actors:      actors,
		Genres:      genres,
	}
}
