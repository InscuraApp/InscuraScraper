package tvmaze

import (
	"fmt"
	"inscurascraper/model"
	"inscurascraper/provider"
	"net/url"
	"strconv"
)

// TVMaze API response types for people.

type personResponse struct {
	ID       int       `json:"id"`
	URL      string    `json:"url"`
	Name     string    `json:"name"`
	Country  *country  `json:"country"`
	Birthday string    `json:"birthday"`
	Deathday *string   `json:"deathday"`
	Gender   string    `json:"gender"`
	Image    *imageObj `json:"image"`
}

type searchPersonResult struct {
	Score  float64        `json:"score"`
	Person personResponse `json:"person"`
}

// NormalizeActorID implements provider.ActorProvider.
func (t *TVMaze) NormalizeActorID(id string) string {
	if _, err := strconv.Atoi(id); err == nil {
		return id
	}
	return ""
}

// ParseActorIDFromURL implements provider.ActorProvider.
func (t *TVMaze) ParseActorIDFromURL(rawURL string) (string, error) {
	return parseIDFromURL(rawURL)
}

// GetActorInfoByID implements provider.ActorProvider.
func (t *TVMaze) GetActorInfoByID(id string) (*model.ActorInfo, error) {
	apiURL := fmt.Sprintf("%s/people/%s", apiBaseURL, id)

	resp := &personResponse{}
	if err := t.apiGet(apiURL, resp); err != nil {
		return nil, err
	}

	if resp.ID == 0 {
		return nil, provider.ErrInfoNotFound
	}

	sid := strconv.Itoa(resp.ID)

	info := &model.ActorInfo{
		ID:       sid,
		Name:     resp.Name,
		Provider: t.Name(),
		Homepage: resp.URL,
		Aliases:  []string{},
		Images:   []string{},
	}

	info.Birthday = parseDate(resp.Birthday)

	if resp.Country != nil {
		info.Nationality = resp.Country.Name
	}

	if resp.Image != nil {
		if resp.Image.Original != "" {
			info.Images = append(info.Images, resp.Image.Original)
		}
		if resp.Image.Medium != "" && resp.Image.Medium != resp.Image.Original {
			info.Images = append(info.Images, resp.Image.Medium)
		}
	}

	return info, nil
}

// GetActorInfoByURL implements provider.ActorProvider.
func (t *TVMaze) GetActorInfoByURL(rawURL string) (*model.ActorInfo, error) {
	id, err := t.ParseActorIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return t.GetActorInfoByID(id)
}

// SearchActor implements provider.ActorSearcher.
func (t *TVMaze) SearchActor(keyword string) ([]*model.ActorSearchResult, error) {
	apiURL := fmt.Sprintf("%s/search/people?q=%s", apiBaseURL, url.QueryEscape(keyword))

	var resp []searchPersonResult
	if err := t.apiGet(apiURL, &resp); err != nil {
		return nil, err
	}

	var results []*model.ActorSearchResult
	for _, r := range resp {
		p := r.Person
		sid := strconv.Itoa(p.ID)

		res := &model.ActorSearchResult{
			ID:       sid,
			Name:     p.Name,
			Provider: t.Name(),
			Homepage: p.URL,
			Images:   []string{},
		}

		if p.Image != nil && p.Image.Original != "" {
			res.Images = append(res.Images, p.Image.Original)
		}

		results = append(results, res)
	}

	if results == nil {
		return nil, provider.ErrInfoNotFound
	}

	return results, nil
}
