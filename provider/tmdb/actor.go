package tmdb

import (
	"fmt"
	"net/url"
	"strconv"

	"inscurascraper/model"
	"inscurascraper/provider"
)

// TMDB API response types for persons.

type searchPersonResponse struct {
	Page         int            `json:"page"`
	Results      []personResult `json:"results"`
	TotalResults int            `json:"total_results"`
	TotalPages   int            `json:"total_pages"`
}

type personResult struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ProfilePath string `json:"profile_path"`
}

type personDetailResponse struct {
	ID           int          `json:"id"`
	Name         string       `json:"name"`
	Biography    string       `json:"biography"`
	Birthday     string       `json:"birthday"`
	Deathday     string       `json:"deathday"`
	PlaceOfBirth string       `json:"place_of_birth"`
	ProfilePath  string       `json:"profile_path"`
	AlsoKnownAs  []string     `json:"also_known_as"`
	Gender       int          `json:"gender"`
	Images       personImages `json:"images"`
}

type personImages struct {
	Profiles []imageDetail `json:"profiles"`
}

type imageDetail struct {
	FilePath string `json:"file_path"`
}

// NormalizeActorID implements provider.ActorProvider.
func (t *TMDB) NormalizeActorID(id string) string {
	if _, err := strconv.Atoi(id); err == nil {
		return id
	}
	return ""
}

// ParseActorIDFromURL implements provider.ActorProvider.
func (t *TMDB) ParseActorIDFromURL(rawURL string) (string, error) {
	return parseIDFromPath(rawURL)
}

// GetActorInfoByID implements provider.ActorProvider.
func (t *TMDB) GetActorInfoByID(id string) (*model.ActorInfo, error) {

	apiURL := fmt.Sprintf("%s/person/%s?language=%s&append_to_response=images",
		apiBaseURL, id, t.resolveLanguage())

	resp := &personDetailResponse{}
	if err := t.apiGet(apiURL, resp); err != nil {
		return nil, err
	}

	sid := strconv.Itoa(resp.ID)

	info := &model.ActorInfo{
		ID:       sid,
		Name:     resp.Name,
		Provider: t.Name(),
		Homepage: fmt.Sprintf(personPageURL, sid),
		Summary:  resp.Biography,
		Aliases:  []string{},
		Images:   []string{},
	}

	info.Nationality = resp.PlaceOfBirth
	info.Birthday = parseDate(resp.Birthday)

	if resp.AlsoKnownAs != nil {
		info.Aliases = resp.AlsoKnownAs
	}

	// Primary profile image first.
	if resp.ProfilePath != "" {
		info.Images = append(info.Images, imageURL(profileOriginal, resp.ProfilePath))
	}
	// Additional profile images (limit to 10).
	for i, img := range resp.Images.Profiles {
		if i >= 10 {
			break
		}
		u := imageURL(profileOriginal, img.FilePath)
		if u != "" && u != imageURL(profileOriginal, resp.ProfilePath) {
			info.Images = append(info.Images, u)
		}
	}

	return info, nil
}

// GetActorInfoByURL implements provider.ActorProvider.
func (t *TMDB) GetActorInfoByURL(rawURL string) (*model.ActorInfo, error) {
	id, err := t.ParseActorIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return t.GetActorInfoByID(id)
}

// SearchActor implements provider.ActorSearcher.
func (t *TMDB) SearchActor(keyword string) ([]*model.ActorSearchResult, error) {

	apiURL := fmt.Sprintf("%s/search/person?language=%s&query=%s",
		apiBaseURL, t.resolveLanguage(), url.QueryEscape(keyword))

	resp := &searchPersonResponse{}
	if err := t.apiGet(apiURL, resp); err != nil {
		return nil, err
	}

	var results []*model.ActorSearchResult
	for _, r := range resp.Results {
		sid := strconv.Itoa(r.ID)

		res := &model.ActorSearchResult{
			ID:       sid,
			Name:     r.Name,
			Provider: t.Name(),
			Homepage: fmt.Sprintf(personPageURL, sid),
			Images:   []string{},
		}

		if r.ProfilePath != "" {
			res.Images = append(res.Images, imageURL(profileOriginal, r.ProfilePath))
		}

		results = append(results, res)
	}

	if results == nil {
		return nil, provider.ErrInfoNotFound
	}

	return results, nil
}
