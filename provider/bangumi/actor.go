package bangumi

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"

	"inscurascraper/model"
	"inscurascraper/provider"
)

var (
	_ provider.ActorProvider = (*Bangumi)(nil)
	_ provider.ActorSearcher = (*Bangumi)(nil)
)

type person struct {
	ID        int           `json:"id"`
	Name      string        `json:"name"`
	Type      int           `json:"type"`
	Career    []string      `json:"career"`
	Images    bgmImages     `json:"images"`
	Summary   string        `json:"summary"`
	BloodType int           `json:"blood_type"`
	BornOn    string        `json:"born_on"`
	BirthYear *int          `json:"birth_year"`
	BirthMon  *int          `json:"birth_mon"`
	BirthDay  *int          `json:"birth_day"`
	Gender    string        `json:"gender"`
	Infobox   []infoboxItem `json:"infobox"`
}

type searchPersonsFilter struct {
	Career []string `json:"career,omitempty"`
}

type searchPersonsRequest struct {
	Keyword string              `json:"keyword"`
	Filter  searchPersonsFilter `json:"filter"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
}

type searchPersonsResponse struct {
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Data   []person `json:"data"`
}

func (b *Bangumi) NormalizeActorID(id string) string {
	if _, err := strconv.Atoi(id); err != nil {
		return ""
	}
	return id
}

func (b *Bangumi) ParseActorIDFromURL(rawURL string) (string, error) {
	return parseIDFromURL(rawURL, "person")
}

func (b *Bangumi) GetActorInfoByID(id string) (*model.ActorInfo, error) {
	if b.NormalizeActorID(id) == "" {
		return nil, provider.ErrInvalidID
	}

	var p person
	if err := b.apiGet(fmt.Sprintf("%s/v0/persons/%s", apiBase, id), &p); err != nil {
		return nil, err
	}
	if p.ID == 0 {
		return nil, provider.ErrInfoNotFound
	}

	return personToActorInfo(&p), nil
}

func (b *Bangumi) GetActorInfoByURL(rawURL string) (*model.ActorInfo, error) {
	id, err := b.ParseActorIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return b.GetActorInfoByID(id)
}

func (b *Bangumi) SearchActor(keyword string) ([]*model.ActorSearchResult, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, provider.ErrInvalidKeyword
	}

	req := searchPersonsRequest{
		Keyword: keyword,
		Filter:  searchPersonsFilter{},
		Limit:   25,
		Offset:  0,
	}
	var resp searchPersonsResponse
	if err := b.apiPost(fmt.Sprintf("%s/v0/search/persons", apiBase), req, &resp); err != nil {
		return nil, err
	}

	var results []*model.ActorSearchResult
	for i := range resp.Data {
		r := personToSearchResult(&resp.Data[i])
		if r.IsValid() {
			results = append(results, r)
		}
	}
	return results, nil
}

func personBirthday(p *person) datatypes.Date {
	if p.BornOn != "" {
		return parseDate(p.BornOn)
	}
	if p.BirthYear != nil && p.BirthMon != nil && p.BirthDay != nil {
		t := time.Date(*p.BirthYear, time.Month(*p.BirthMon), *p.BirthDay, 0, 0, 0, 0, time.UTC)
		return datatypes.Date(t)
	}
	return datatypes.Date(time.Time{})
}

func personAliases(p *person) []string {
	var aliases []string
	if cn := infoboxGet(p.Infobox, "简体中文名"); cn != "" {
		aliases = append(aliases, cn)
	}
	aliases = append(aliases, infoboxGetAll(p.Infobox, "别名")...)
	return aliases
}

func personToSearchResult(p *person) *model.ActorSearchResult {
	id := strconv.Itoa(p.ID)
	return &model.ActorSearchResult{
		ID:       id,
		Name:     p.Name,
		Provider: Name,
		Homepage: fmt.Sprintf("%sperson/%d", baseURL, p.ID),
		Aliases:  personAliases(p),
		Images:   extractImageURLs(p.Images),
	}
}

func personToActorInfo(p *person) *model.ActorInfo {
	id := strconv.Itoa(p.ID)
	return &model.ActorInfo{
		ID:          id,
		Name:        p.Name,
		Provider:    Name,
		Homepage:    fmt.Sprintf("%sperson/%d", baseURL, p.ID),
		Summary:     p.Summary,
		BloodType:   bloodTypeStr(p.BloodType),
		Nationality: infoboxGet(p.Infobox, "出生地"),
		Aliases:     personAliases(p),
		Images:      extractImageURLs(p.Images),
		Birthday:    personBirthday(p),
	}
}
