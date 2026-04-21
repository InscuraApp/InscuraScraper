package tvdb

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"inscurascraper/model"
	"inscurascraper/provider"
)

// TVDB API response types for people.

type personExtended struct {
	ID                   int         `json:"id"`
	Name                 string      `json:"name"`
	Slug                 string      `json:"slug"`
	Image                string      `json:"image"`
	Birth                string      `json:"birth"`
	Death                string      `json:"death"`
	BirthPlace           string      `json:"birthPlace"`
	Gender               int         `json:"gender"`
	Score                int         `json:"score"`
	Biographies          []biography `json:"biographies"`
	Characters           []character `json:"characters"`
	Aliases              []alias     `json:"aliases"`
	RemoteIDs            []remoteID  `json:"remoteIds"`
	NameTranslations     []string    `json:"nameTranslations"`
	OverviewTranslations []string    `json:"overviewTranslations"`
	Translations         *translations `json:"translations"`
}

type alias struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}

// NormalizeActorID implements provider.ActorProvider.
func (t *TVDB) NormalizeActorID(id string) string {
	if _, err := strconv.Atoi(id); err == nil {
		return id
	}
	return ""
}

// ParseActorIDFromURL implements provider.ActorProvider.
func (t *TVDB) ParseActorIDFromURL(rawURL string) (string, error) {
	return parsePersonIDFromURL(rawURL)
}

// GetActorInfoByID implements provider.ActorProvider.
func (t *TVDB) GetActorInfoByID(id string) (*model.ActorInfo, error) {

	apiURL := fmt.Sprintf("%s/people/%s/extended", apiBaseURL, id)

	var resp apiResponse[personExtended]
	if err := t.apiGet(apiURL, &resp); err != nil {
		return nil, err
	}

	p := resp.Data
	if p.ID == 0 {
		return nil, provider.ErrInfoNotFound
	}

	sid := strconv.Itoa(p.ID)

	info := &model.ActorInfo{
		ID:       sid,
		Name:     p.Name,
		Provider: t.Name(),
		Homepage: fmt.Sprintf(personPageURL, sid),
		Aliases:  []string{},
		Images:   []string{},
	}

	// Name translation for resolved language.
	lang := t.resolveLanguage()
	if p.Translations != nil {
		if trName, _ := findTranslation(p.Translations.NameTranslations, lang); trName != "" {
			info.Name = trName
		}
	}

	// Biography: prefer resolved language, fallback to English.
	for _, bio := range p.Biographies {
		if bio.Language == lang {
			info.Summary = bio.Biography
			break
		}
	}
	if info.Summary == "" {
		for _, bio := range p.Biographies {
			if bio.Language == "eng" {
				info.Summary = bio.Biography
				break
			}
		}
	}

	info.Nationality = p.BirthPlace
	info.Birthday = parseDate(p.Birth)

	// Aliases.
	for _, a := range p.Aliases {
		if a.Name != "" && a.Name != p.Name {
			info.Aliases = append(info.Aliases, a.Name)
		}
	}

	// Primary image.
	if p.Image != "" {
		info.Images = append(info.Images, p.Image)
	}

	// Character images as additional images (limit 10).
	seen := map[string]bool{}
	if p.Image != "" {
		seen[p.Image] = true
	}
	for _, ch := range p.Characters {
		if len(info.Images) >= 10 {
			break
		}
		imgURL := ch.PersonImgURL
		if imgURL == "" || !strings.HasPrefix(imgURL, "http") || seen[imgURL] {
			continue
		}
		info.Images = append(info.Images, imgURL)
		seen[imgURL] = true
	}

	return info, nil
}

// GetActorInfoByURL implements provider.ActorProvider.
func (t *TVDB) GetActorInfoByURL(rawURL string) (*model.ActorInfo, error) {
	id, err := t.ParseActorIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return t.GetActorInfoByID(id)
}

// SearchActor implements provider.ActorSearcher.
func (t *TVDB) SearchActor(keyword string) ([]*model.ActorSearchResult, error) {

	apiURL := fmt.Sprintf("%s/search?query=%s&type=people",
		apiBaseURL, url.QueryEscape(keyword))

	var resp apiResponse[[]searchResult]
	if err := t.apiGet(apiURL, &resp); err != nil {
		return nil, err
	}

	var results []*model.ActorSearchResult
	for _, r := range resp.Data {
		if r.Type != "person" && r.Type != "people" {
			continue
		}

		tvdbID := r.TvdbID
		if tvdbID == "" {
			tvdbID = r.ObjectID
		}

		name := r.Name
		if trName, ok := r.Translations[t.resolveLanguage()]; ok && trName != "" {
			name = trName
		}

		res := &model.ActorSearchResult{
			ID:       tvdbID,
			Name:     name,
			Provider: t.Name(),
			Homepage: fmt.Sprintf(personPageURL, tvdbID),
			Images:   []string{},
		}

		if r.ImageURL != "" {
			res.Images = append(res.Images, r.ImageURL)
		} else if r.Thumbnail != "" {
			res.Images = append(res.Images, r.Thumbnail)
		}

		results = append(results, res)
	}

	if results == nil {
		return nil, provider.ErrInfoNotFound
	}

	return results, nil
}
