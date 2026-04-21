package rank

import (
	"testing"
	"time"

	"inscurascraper/model"

	"gorm.io/datatypes"
)

func date(y, m, d int) datatypes.Date {
	return datatypes.Date(time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC))
}

func TestRankMovies_YearAndTitleWin(t *testing.T) {
	// Simulates Spider-Man No Way Home search with year=2021.
	results := []*model.MovieSearchResult{
		{ID: "9613", Provider: "TMDB", Title: "Spider", ReleaseDate: date(2002, 11, 6)},
		{ID: "104308", Provider: "TMDB", Title: "The Spider Labyrinth", ReleaseDate: date(1988, 8, 25)},
		{ID: "634649", Provider: "TMDB", Title: "Spider-Man: No Way Home", ReleaseDate: date(2021, 12, 15)},
	}
	priority := func(string) float64 { return 1000 }
	got := RankMovies(results, MovieCriteria{
		Keyword:     "Spider-Man No Way Home",
		Year:        2021,
		MaxPriority: 1000,
		PriorityOf:  priority,
	})
	if got[0].ID != "634649" {
		t.Fatalf("expected 634649 first, got: %v", ids(got))
	}
}

func TestRankMovies_RoundRobinPromotesEachProvider(t *testing.T) {
	results := []*model.MovieSearchResult{
		{ID: "A1", Provider: "TMDB", Title: "foo alpha"},
		{ID: "A2", Provider: "TMDB", Title: "foo alpha 2"},
		{ID: "A3", Provider: "TMDB", Title: "foo alpha 3"},
		{ID: "B1", Provider: "TVDB", Title: "foo alpha"}, // same score as A1
		{ID: "C1", Provider: "AniList", Title: "foo alpha"},
	}
	priority := func(string) float64 { return 1000 }
	got := RankMovies(results, MovieCriteria{
		Keyword:     "foo alpha",
		MaxPriority: 1000,
		PriorityOf:  priority,
	})
	// The first three slots should be three different providers.
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		seen[got[i].Provider] = true
	}
	if len(seen) != 3 {
		t.Fatalf("expected top-3 to cover 3 providers, got %v from: %v", seen, ids(got))
	}
}

func TestRankMovies_YearOffByOneStillRanksHigh(t *testing.T) {
	results := []*model.MovieSearchResult{
		{ID: "new", Provider: "TMDB", Title: "Unrelated Title", ReleaseDate: date(2021, 1, 1)},
		{ID: "want", Provider: "TMDB", Title: "Spider-Man Into The Spider-Verse", ReleaseDate: date(2018, 12, 14)},
	}
	got := RankMovies(results, MovieCriteria{
		Keyword:     "Spider-Man",
		Year:        2019, // off by one from 2018
		MaxPriority: 1000,
		PriorityOf:  func(string) float64 { return 1000 },
	})
	if got[0].ID != "want" {
		t.Fatalf("expected 'want' first (year off-by-one + title match), got: %v", ids(got))
	}
}

func TestRankMovies_UnknownYear_TitleWins(t *testing.T) {
	// When the searched year is 0, tier-1 collapses and title drives ordering.
	results := []*model.MovieSearchResult{
		{ID: "other", Provider: "TMDB", Title: "Something Else Entirely"},
		{ID: "hit", Provider: "TMDB", Title: "Fight Club"},
	}
	got := RankMovies(results, MovieCriteria{
		Keyword:     "Fight Club",
		MaxPriority: 1000,
		PriorityOf:  func(string) float64 { return 1000 },
	})
	if got[0].ID != "hit" {
		t.Fatalf("expected 'hit' first, got: %v", ids(got))
	}
}

func TestRankActors_RoundRobin(t *testing.T) {
	results := []*model.ActorSearchResult{
		{ID: "A1", Provider: "TMDB", Name: "John Smith"},
		{ID: "A2", Provider: "TMDB", Name: "John Smith"},
		{ID: "B1", Provider: "TVDB", Name: "John Smith"},
	}
	got := RankActors(results, ActorCriteria{
		Keyword:     "John Smith",
		MaxPriority: 1000,
		PriorityOf:  func(string) float64 { return 1000 },
	})
	if got[0].Provider == got[1].Provider {
		t.Fatalf("expected first two rows to be from different providers, got: %v", actorIDs(got))
	}
}

func TestRankMovies_ChineseLanguagePrefersHanTitle(t *testing.T) {
	results := []*model.MovieSearchResult{
		{ID: "en", Provider: "TMDB", Title: "Fight Club", ReleaseDate: date(1999, 10, 15)},
		{ID: "zh", Provider: "TVDB", Title: "搏击俱乐部", ReleaseDate: date(1999, 10, 15)},
	}
	got := RankMovies(results, MovieCriteria{
		Keyword:     "Fight Club",
		Year:        1999,
		Language:    "zh-CN",
		MaxPriority: 1000,
		PriorityOf:  func(string) float64 { return 1000 },
	})
	if got[0].ID != "zh" {
		t.Fatalf("zh-CN request should prefer the Han-script title, got: %v", ids(got))
	}
}

func TestRankMovies_JapaneseLanguagePrefersKanaTitle(t *testing.T) {
	results := []*model.MovieSearchResult{
		{ID: "en", Provider: "TMDB", Title: "Cowboy Bebop", ReleaseDate: date(1998, 4, 3)},
		{ID: "jp", Provider: "AniList", Title: "カウボーイビバップ", ReleaseDate: date(1998, 4, 3)},
	}
	got := RankMovies(results, MovieCriteria{
		Keyword:     "Cowboy Bebop",
		Year:        1998,
		Language:    "ja-JP",
		MaxPriority: 1000,
		PriorityOf:  func(string) float64 { return 1000 },
	})
	if got[0].ID != "jp" {
		t.Fatalf("ja-JP request should prefer the kana title, got: %v", ids(got))
	}
}

func TestRankMovies_LanguageCannotBeatYear(t *testing.T) {
	// A zh-CN request still shouldn't promote a zh title from the wrong year
	// above an en-US title from the correct year.
	results := []*model.MovieSearchResult{
		{ID: "wrong-year-zh", Provider: "TVDB", Title: "搏击俱乐部", ReleaseDate: date(2010, 1, 1)},
		{ID: "right-year-en", Provider: "TMDB", Title: "Fight Club", ReleaseDate: date(1999, 10, 15)},
	}
	got := RankMovies(results, MovieCriteria{
		Keyword:     "Fight Club",
		Year:        1999,
		Language:    "zh-CN",
		MaxPriority: 1000,
		PriorityOf:  func(string) float64 { return 1000 },
	})
	if got[0].ID != "right-year-en" {
		t.Fatalf("year should dominate language, got: %v", ids(got))
	}
}

func TestRankMovies_NoLanguageRequested_NoLanguageBias(t *testing.T) {
	// Without X-Is-Language, language tier is inert and the English title
	// (better Jaro-Winkler match) should win.
	results := []*model.MovieSearchResult{
		{ID: "zh", Provider: "TVDB", Title: "搏击俱乐部", ReleaseDate: date(1999, 10, 15)},
		{ID: "en", Provider: "TMDB", Title: "Fight Club", ReleaseDate: date(1999, 10, 15)},
	}
	got := RankMovies(results, MovieCriteria{
		Keyword:     "Fight Club",
		Year:        1999,
		MaxPriority: 1000,
		PriorityOf:  func(string) float64 { return 1000 },
	})
	if got[0].ID != "en" {
		t.Fatalf("without language, English title should win on similarity, got: %v", ids(got))
	}
}

func ids(xs []*model.MovieSearchResult) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = x.Provider + ":" + x.ID
	}
	return out
}

func actorIDs(xs []*model.ActorSearchResult) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = x.Provider + ":" + x.ID
	}
	return out
}
