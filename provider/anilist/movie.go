package anilist

import (
	"fmt"
	"strconv"
	"strings"

	"inscurascraper/model"
	"inscurascraper/provider"
)

// GraphQL queries for media. The %s placeholder receives the AniList
// voice-actor language enum resolved from the per-request X-Is-Language header.

const mediaDetailQueryTmpl = `
query ($id: Int, $type: MediaType) {
  Media(id: $id, type: $type) {
    id title { romaji english native }
    description(asHtml: false)
    genres
    coverImage { extraLarge large medium }
    bannerImage
    episodes duration
    startDate { year month day }
    season seasonYear status format
    averageScore meanScore
    studios { nodes { name isAnimationStudio } }
    staff(sort: RELEVANCE, perPage: 5) {
      nodes { id name { full native } }
    }
    characters(sort: ROLE, perPage: 15) {
      edges {
        role
        voiceActors(language: %s) { id name { full } }
        node { id name { full native } image { large } }
      }
    }
    trailer { id site thumbnail }
    synonyms
  }
}
`

const mediaSearchQuery = `
query ($search: String, $type: MediaType, $page: Int, $perPage: Int) {
  Page(page: $page, perPage: $perPage) {
    media(search: $search, type: $type, sort: SEARCH_MATCH) {
      id title { romaji english native }
      coverImage { large medium }
      bannerImage
      averageScore format episodes
      startDate { year month day }
    }
  }
}
`

// Response types.

type mediaDetailResponse struct {
	Data struct {
		Media *mediaDetail `json:"Media"`
	} `json:"data"`
}

type mediaDetail struct {
	ID           int         `json:"id"`
	Title        mediaTitle  `json:"title"`
	Description  string      `json:"description"`
	Genres       []string    `json:"genres"`
	CoverImage   *coverImage `json:"coverImage"`
	BannerImage  string      `json:"bannerImage"`
	Episodes     *int        `json:"episodes"`
	Duration     *int        `json:"duration"`
	StartDate    *fuzzyDate  `json:"startDate"`
	Season       *string     `json:"season"`
	SeasonYear   *int        `json:"seasonYear"`
	Status       string      `json:"status"`
	Format       string      `json:"format"`
	AverageScore *int        `json:"averageScore"`
	MeanScore    *int        `json:"meanScore"`
	Studios      struct {
		Nodes []studioNode `json:"nodes"`
	} `json:"studios"`
	Staff struct {
		Nodes []struct {
			ID   int       `json:"id"`
			Name staffName `json:"name"`
		} `json:"nodes"`
	} `json:"staff"`
	Characters struct {
		Edges []struct {
			Role string `json:"role"`
			VoiceActors []struct {
				ID   int `json:"id"`
				Name struct {
					Full string `json:"full"`
				} `json:"name"`
			} `json:"voiceActors"`
			Node struct {
				ID   int `json:"id"`
				Name struct {
					Full   string `json:"full"`
					Native string `json:"native"`
				} `json:"name"`
				Image *characterImage `json:"image"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"characters"`
	Trailer  *trailerObj `json:"trailer"`
	Synonyms []string    `json:"synonyms"`
}

type mediaSearchResponse struct {
	Data struct {
		Page struct {
			Media []mediaSearchItem `json:"media"`
		} `json:"Page"`
	} `json:"data"`
}

type mediaSearchItem struct {
	ID           int         `json:"id"`
	Title        mediaTitle  `json:"title"`
	CoverImage   *coverImage `json:"coverImage"`
	BannerImage  string      `json:"bannerImage"`
	AverageScore *int        `json:"averageScore"`
	Format       string      `json:"format"`
	Episodes     *int        `json:"episodes"`
	StartDate    *fuzzyDate  `json:"startDate"`
}

// NormalizeMovieID implements provider.MovieProvider.
func (a *AniList) NormalizeMovieID(id string) string {
	kind, numID := parseID(id)
	if _, err := strconv.Atoi(numID); err != nil {
		return ""
	}
	return kind + ":" + numID
}

// ParseMovieIDFromURL implements provider.MovieProvider.
func (a *AniList) ParseMovieIDFromURL(rawURL string) (string, error) {
	return parseIDFromURL(rawURL)
}

// GetMovieInfoByID implements provider.MovieProvider.
func (a *AniList) GetMovieInfoByID(id string) (*model.MovieInfo, error) {
	kind, numID := parseID(id)
	numericID, err := strconv.Atoi(numID)
	if err != nil {
		return nil, provider.ErrInvalidID
	}

	mediaType := "ANIME"
	if kind == "manga" {
		mediaType = "MANGA"
	}

	var resp mediaDetailResponse
	query := fmt.Sprintf(mediaDetailQueryTmpl, a.resolveVoiceActorLanguage())
	if err := a.graphqlQuery(query, map[string]any{
		"id":   numericID,
		"type": mediaType,
	}, &resp); err != nil {
		return nil, err
	}

	m := resp.Data.Media
	if m == nil {
		return nil, provider.ErrInfoNotFound
	}

	sid := strconv.Itoa(m.ID)
	prefixedID := kind + ":" + sid

	pageURL := animePageURL
	if kind == "manga" {
		pageURL = mangaPageURL
	}

	info := &model.MovieInfo{
		ID:            prefixedID,
		Number:        sid,
		Provider:      a.Name(),
		Homepage:      fmt.Sprintf(pageURL, sid),
		Actors:        []string{},
		PreviewImages: []string{},
		Genres:        m.Genres,
	}

	if info.Genres == nil {
		info.Genres = []string{}
	}

	// Title: picked according to X-Is-Language (English/Native/Romaji priority).
	info.Title = a.resolveTitle(m.Title)

	info.Summary = cleanDescription(m.Description)

	if m.AverageScore != nil {
		info.Score = float64(*m.AverageScore) / 10.0 // Convert 0-100 to 0-10
	}
	if m.Duration != nil {
		info.Runtime = *m.Duration
	}
	info.ReleaseDate = parseDate(m.StartDate)

	// Cover image.
	if m.CoverImage != nil {
		info.ThumbURL = m.CoverImage.Medium
		info.BigThumbURL = m.CoverImage.ExtraLarge
		if info.BigThumbURL == "" {
			info.BigThumbURL = m.CoverImage.Large
		}
	}

	// Banner as cover.
	if m.BannerImage != "" {
		info.CoverURL = m.BannerImage
		info.BigCoverURL = m.BannerImage
	} else if m.CoverImage != nil {
		info.CoverURL = m.CoverImage.ExtraLarge
		info.BigCoverURL = info.CoverURL
	}

	// Director from staff.
	for _, s := range m.Staff.Nodes {
		if s.Name.Full != "" {
			info.Director = s.Name.Full
			break
		}
	}

	// Actors: voice actors from character edges.
	seen := map[string]bool{}
	for _, edge := range m.Characters.Edges {
		for _, va := range edge.VoiceActors {
			if va.Name.Full != "" && !seen[va.Name.Full] && len(info.Actors) < 15 {
				info.Actors = append(info.Actors, va.Name.Full)
				seen[va.Name.Full] = true
			}
		}
	}
	// If no voice actors, use character names.
	if len(info.Actors) == 0 {
		for _, edge := range m.Characters.Edges {
			if edge.Node.Name.Full != "" && len(info.Actors) < 15 {
				info.Actors = append(info.Actors, edge.Node.Name.Full)
			}
		}
	}

	// Maker: animation studio.
	for _, studio := range m.Studios.Nodes {
		if studio.IsAnimationStudio {
			info.Maker = studio.Name
			break
		}
	}
	if info.Maker == "" && len(m.Studios.Nodes) > 0 {
		info.Maker = m.Studios.Nodes[0].Name
	}

	// Trailer.
	if m.Trailer != nil && m.Trailer.Site == "youtube" {
		info.PreviewVideoURL = "https://www.youtube.com/watch?v=" + m.Trailer.ID
	}

	return info, nil
}

// GetMovieInfoByURL implements provider.MovieProvider.
func (a *AniList) GetMovieInfoByURL(rawURL string) (*model.MovieInfo, error) {
	id, err := a.ParseMovieIDFromURL(rawURL)
	if err != nil {
		return nil, err
	}
	return a.GetMovieInfoByID(id)
}

// NormalizeMovieKeyword implements provider.MovieSearcher.
// The engine already strips release-name artefacts before calling us.
func (a *AniList) NormalizeMovieKeyword(keyword string) string {
	return keyword
}

// SearchMovie implements provider.MovieSearcher.
func (a *AniList) SearchMovie(keyword string) ([]*model.MovieSearchResult, error) {
	// Search both anime and manga, combine results.
	var allResults []*model.MovieSearchResult

	for _, mediaType := range []string{"ANIME", "MANGA"} {
		var resp mediaSearchResponse
		if err := a.graphqlQuery(mediaSearchQuery, map[string]any{
			"search":  keyword,
			"type":    mediaType,
			"page":    1,
			"perPage": 10,
		}, &resp); err != nil {
			continue
		}

		prefix := animePrefix
		pageURLFmt := animePageURL
		if mediaType == "MANGA" {
			prefix = mangaPrefix
			pageURLFmt = mangaPageURL
		}

		for _, m := range resp.Data.Page.Media {
			sid := strconv.Itoa(m.ID)

			title := a.resolveTitle(m.Title)

			var thumbURL, coverURL string
			if m.CoverImage != nil {
				thumbURL = m.CoverImage.Medium
				coverURL = m.CoverImage.Large
			}
			if m.BannerImage != "" {
				coverURL = m.BannerImage
			}

			var score float64
			if m.AverageScore != nil {
				score = float64(*m.AverageScore) / 10.0
			}

			allResults = append(allResults, &model.MovieSearchResult{
				ID:          prefix + sid,
				Number:      sid,
				Title:       title + formatSuffix(m.Format),
				Provider:    a.Name(),
				Homepage:    fmt.Sprintf(pageURLFmt, sid),
				ThumbURL:    thumbURL,
				CoverURL:    coverURL,
				Score:       score,
				ReleaseDate: parseDate(m.StartDate),
			})
		}
	}

	if allResults == nil {
		return nil, provider.ErrInfoNotFound
	}

	return allResults, nil
}

// formatSuffix returns a display suffix for the media format.
func formatSuffix(format string) string {
	switch format {
	case "TV":
		return ""
	case "MOVIE":
		return " [Movie]"
	case "OVA":
		return " [OVA]"
	case "ONA":
		return " [ONA]"
	case "SPECIAL":
		return " [Special]"
	case "MANGA":
		return " [Manga]"
	case "NOVEL":
		return " [Novel]"
	case "ONE_SHOT":
		return " [One Shot]"
	default:
		if format != "" {
			return " [" + strings.ReplaceAll(format, "_", " ") + "]"
		}
		return ""
	}
}
