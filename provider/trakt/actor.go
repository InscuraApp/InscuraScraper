package trakt

import (
	"fmt"
	"inscurascraper/model"
	"inscurascraper/provider"
	"net/url"
	"path"
	"strings"
)

// Trakt API response types for people.

type searchPersonItem struct {
	Type   string       `json:"type"`
	Score  float64      `json:"score"`
	Person personDetail `json:"person"`
}

type personDetail struct {
	Name       string   `json:"name"`
	IDs        traktIDs `json:"ids"`
	Biography  string   `json:"biography"`
	Birthday   string   `json:"birthday"`
	Birthplace string   `json:"birthplace"`
	Death      string   `json:"death"`
	Homepage   string   `json:"homepage"`
}

// NormalizeActorID implements provider.ActorProvider.
func (t *Trakt) NormalizeActorID(id string) string {
	lower := strings.ToLower(id)
	if slugRe.MatchString(lower) {
		return lower
	}
	return ""
}

// ParseActorIDFromURL implements provider.ActorProvider.
func (t *Trakt) ParseActorIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", provider.ErrInvalidURL
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) < 2 || segments[0] != "people" {
		return "", provider.ErrInvalidURL
	}
	slug := path.Base(u.Path)
	if t.NormalizeActorID(slug) == "" {
		return "", provider.ErrInvalidURL
	}
	return slug, nil
}

// GetActorInfoByID implements provider.ActorProvider.
func (t *Trakt) GetActorInfoByID(id string) (*model.ActorInfo, error) {
	detail := &personDetail{}
	if err := t.apiGet(fmt.Sprintf("%s/people/%s?extended=full", apiBaseURL, id), detail); err != nil {
		return nil, err
	}
	if detail.Name == "" {
		return nil, provider.ErrInfoNotFound
	}

	slug := detail.IDs.Slug
	if slug == "" {
		slug = id
	}

	info := &model.ActorInfo{
		ID:          slug,
		Name:        detail.Name,
		Provider:    t.Name(),
		Homepage:    fmt.Sprintf(personPageURL, slug),
		Summary:     detail.Biography,
		Nationality: detail.Birthplace,
		Birthday:    parseDate(detail.Birthday),
		Aliases:     []string{},
		Images:      []string{},
	}

	return info, nil
}

// GetActorInfoByURL implements provider.ActorProvider.
func (t *Trakt) GetActorInfoByURL(rawURL string) (*model.ActorInfo, error) {
	id, err := t.ParseActorIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return t.GetActorInfoByID(id)
}

// SearchActor implements provider.ActorSearcher.
func (t *Trakt) SearchActor(keyword string) ([]*model.ActorSearchResult, error) {
	apiURL := fmt.Sprintf("%s/search/person?query=%s&extended=full", apiBaseURL, url.QueryEscape(keyword))

	var items []searchPersonItem
	if err := t.apiGet(apiURL, &items); err != nil {
		return nil, err
	}

	var results []*model.ActorSearchResult
	for _, item := range items {
		p := item.Person
		slug := p.IDs.Slug
		if slug == "" {
			continue
		}
		results = append(results, &model.ActorSearchResult{
			ID:       slug,
			Name:     p.Name,
			Provider: t.Name(),
			Homepage: fmt.Sprintf(personPageURL, slug),
			Images:   []string{},
		})
	}

	if results == nil {
		return nil, provider.ErrInfoNotFound
	}
	return results, nil
}
