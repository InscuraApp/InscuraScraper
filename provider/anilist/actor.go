package anilist

import (
	"fmt"
	"strconv"

	"inscurascraper/model"
	"inscurascraper/provider"
)

// GraphQL queries for staff (mapped to ActorProvider).

const staffDetailQuery = `
query ($id: Int) {
  Staff(id: $id) {
    id
    name { full native userPreferred }
    image { large medium }
    description(asHtml: false)
    primaryOccupations
    gender
    dateOfBirth { year month day }
    dateOfDeath { year month day }
    age
    homeTown
    bloodType
    languageV2
  }
}
`

const staffSearchQuery = `
query ($search: String, $page: Int, $perPage: Int) {
  Page(page: $page, perPage: $perPage) {
    staff(search: $search) {
      id
      name { full native }
      image { large medium }
      primaryOccupations
      homeTown
    }
  }
}
`

// Response types.

type staffDetailResponse struct {
	Data struct {
		Staff *staffDetail `json:"Staff"`
	} `json:"data"`
}

type staffDetail struct {
	ID                  int        `json:"id"`
	Name                staffName  `json:"name"`
	Image               *staffImage `json:"image"`
	Description         string     `json:"description"`
	PrimaryOccupations  []string   `json:"primaryOccupations"`
	Gender              *string    `json:"gender"`
	DateOfBirth         *fuzzyDate `json:"dateOfBirth"`
	DateOfDeath         *fuzzyDate `json:"dateOfDeath"`
	Age                 *int       `json:"age"`
	HomeTown            string     `json:"homeTown"`
	BloodType           *string    `json:"bloodType"`
	LanguageV2          string     `json:"languageV2"`
}

type staffSearchResponse struct {
	Data struct {
		Page struct {
			Staff []staffSearchItem `json:"staff"`
		} `json:"Page"`
	} `json:"data"`
}

type staffSearchItem struct {
	ID                 int        `json:"id"`
	Name               staffName  `json:"name"`
	Image              *staffImage `json:"image"`
	PrimaryOccupations []string   `json:"primaryOccupations"`
	HomeTown           string     `json:"homeTown"`
}

// NormalizeActorID implements provider.ActorProvider.
func (a *AniList) NormalizeActorID(id string) string {
	if _, err := strconv.Atoi(id); err == nil {
		return id
	}
	return ""
}

// ParseActorIDFromURL implements provider.ActorProvider.
func (a *AniList) ParseActorIDFromURL(rawURL string) (string, error) {
	return parseStaffIDFromURL(rawURL)
}

// GetActorInfoByID implements provider.ActorProvider.
func (a *AniList) GetActorInfoByID(id string) (*model.ActorInfo, error) {
	numericID, err := strconv.Atoi(id)
	if err != nil {
		return nil, provider.ErrInvalidID
	}

	var resp staffDetailResponse
	if err := a.graphqlQuery(staffDetailQuery, map[string]any{
		"id": numericID,
	}, &resp); err != nil {
		return nil, err
	}

	s := resp.Data.Staff
	if s == nil {
		return nil, provider.ErrInfoNotFound
	}

	sid := strconv.Itoa(s.ID)

	// Pick primary name based on X-Is-Language. For ja* prefer Native (kanji),
	// otherwise fall back to Full (romanized / English).
	primaryName := s.Name.Full
	aliasName := s.Name.Native
	if isJapanese(a.GetRequestConfig().LanguageOr("")) && s.Name.Native != "" {
		primaryName, aliasName = s.Name.Native, s.Name.Full
	}

	info := &model.ActorInfo{
		ID:       sid,
		Name:     primaryName,
		Provider: a.Name(),
		Homepage: fmt.Sprintf(staffPageURL, sid),
		Summary:  cleanDescription(s.Description),
		Aliases:  []string{},
		Images:   []string{},
	}

	if aliasName != "" && aliasName != primaryName {
		info.Aliases = append(info.Aliases, aliasName)
	}

	info.Birthday = parseDate(s.DateOfBirth)
	info.Nationality = s.HomeTown

	if s.BloodType != nil {
		info.BloodType = *s.BloodType
	}

	// Images.
	if s.Image != nil {
		if s.Image.Large != "" {
			info.Images = append(info.Images, s.Image.Large)
		}
	}

	return info, nil
}

// GetActorInfoByURL implements provider.ActorProvider.
func (a *AniList) GetActorInfoByURL(rawURL string) (*model.ActorInfo, error) {
	id, err := a.ParseActorIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return a.GetActorInfoByID(id)
}

// SearchActor implements provider.ActorSearcher.
func (a *AniList) SearchActor(keyword string) ([]*model.ActorSearchResult, error) {
	var resp staffSearchResponse
	if err := a.graphqlQuery(staffSearchQuery, map[string]any{
		"search":  keyword,
		"page":    1,
		"perPage": 10,
	}, &resp); err != nil {
		return nil, err
	}

	var results []*model.ActorSearchResult
	jp := isJapanese(a.GetRequestConfig().LanguageOr(""))
	for _, s := range resp.Data.Page.Staff {
		sid := strconv.Itoa(s.ID)

		name := s.Name.Full
		if jp && s.Name.Native != "" {
			name = s.Name.Native
		}

		res := &model.ActorSearchResult{
			ID:       sid,
			Name:     name,
			Provider: a.Name(),
			Homepage: fmt.Sprintf(staffPageURL, sid),
			Images:   []string{},
		}

		if s.Image != nil && s.Image.Large != "" {
			res.Images = append(res.Images, s.Image.Large)
		}

		results = append(results, res)
	}

	if results == nil {
		return nil, provider.ErrInfoNotFound
	}

	return results, nil
}
