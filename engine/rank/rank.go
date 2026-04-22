// Package rank provides cross-provider ranking for search results.
//
// The goal is to surface the most relevant hit per provider at the top of
// the merged list so that every provider gets a first-screen chance, while
// still sorting the long tail by relevance.
//
// Scoring is a fused tier model:
//
//	tier1 (year match)        × 1000
//	tier1.5 (language/script) × 100
//	tier2 (title similarity)  × 10
//	tier3 (provider priority) × 1
//
// Tiers are lexicographic (a tier-1 miss can never outrank a tier-1 hit),
// but each tier is a continuous [0..1] score so within-tier ordering is
// meaningful too.
//
// After scoring, results are sorted descending, and a round-robin pass
// lifts the highest-scoring entry per provider to the top, preserving the
// remaining order below. This guarantees that a merged list of, say,
// TMDB + TVDB + AniList + TVMaze + FanartTV shows the single best match
// from each source in the first 5 rows.
package rank

import (
	"inscurascraper/common/comparer"
	"inscurascraper/model"
	"sort"
	"strings"
	"time"
	"unicode"

	"gorm.io/datatypes"
)

// PriorityOf returns the Priority of a provider by name. Engine supplies this
// (the rank package doesn't depend on engine-internal types).
type PriorityOf func(providerName string) float64

// MovieCriteria is the request-side input to movie ranking.
type MovieCriteria struct {
	// Keyword is the cleaned search title (already passed through the release-name parser).
	Keyword string
	// Year is the parser-extracted release year (0 when not available).
	Year int
	// Language is the BCP 47 tag from X-Is-Language (e.g. "zh-CN", "en-US").
	// Empty disables the language tier.
	Language string
	// MaxPriority is the largest Priority among active providers; used to
	// normalise the per-provider tier-3 weight to [0..1]. Pass 0 to disable.
	MaxPriority float64
	// PriorityOf looks up a provider's Priority by name.
	PriorityOf PriorityOf
}

// ActorCriteria is the request-side input to actor ranking.
type ActorCriteria struct {
	Keyword     string
	Language    string
	MaxPriority float64
	PriorityOf  PriorityOf
}

const (
	tier1Weight   = 1000.0 // year match
	tier1_5Weight = 100.0  // language/script match
	tier2Weight   = 10.0   // title similarity + exact phrase
	tier3Weight   = 1.0    // provider priority
)

// ScoreMovie computes the composite score for a single movie result.
func ScoreMovie(r *model.MovieSearchResult, c MovieCriteria) float64 {
	return tier1Weight*yearTier(c.Year, r.ReleaseDate) +
		tier1_5Weight*languageTier(c.Language, r.Title) +
		tier2Weight*titleTier(c.Keyword, r.Title) +
		tier3Weight*priorityTier(r.Provider, c.PriorityOf, c.MaxPriority)
}

// RankMovies sorts a slice of movie results by composite score descending,
// then promotes the top-1 per provider so each provider appears as early
// as possible.
func RankMovies(results []*model.MovieSearchResult, c MovieCriteria) []*model.MovieSearchResult {
	if len(results) == 0 {
		return results
	}
	scored := make([]scoredMovie, len(results))
	for i, r := range results {
		scored[i] = scoredMovie{r: r, s: ScoreMovie(r, c)}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].s > scored[j].s
	})

	out := make([]*model.MovieSearchResult, len(scored))
	for i, sm := range scored {
		out[i] = sm.r
	}
	return promoteTopPerProvider(out, func(r *model.MovieSearchResult) string { return r.Provider })
}

// ScoreActor computes the composite score for a single actor result. Actors
// have no year signal, so only tiers 1.5, 2 and 3 apply.
func ScoreActor(r *model.ActorSearchResult, c ActorCriteria) float64 {
	return tier1_5Weight*languageTier(c.Language, r.Name) +
		tier2Weight*titleTier(c.Keyword, r.Name) +
		tier3Weight*priorityTier(r.Provider, c.PriorityOf, c.MaxPriority)
}

// RankActors sorts actor results and applies round-robin promotion.
func RankActors(results []*model.ActorSearchResult, c ActorCriteria) []*model.ActorSearchResult {
	if len(results) == 0 {
		return results
	}
	scored := make([]scoredActor, len(results))
	for i, r := range results {
		scored[i] = scoredActor{r: r, s: ScoreActor(r, c)}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].s > scored[j].s
	})

	out := make([]*model.ActorSearchResult, len(scored))
	for i, sm := range scored {
		out[i] = sm.r
	}
	return promoteTopPerProvider(out, func(r *model.ActorSearchResult) string { return r.Provider })
}

type scoredMovie struct {
	r *model.MovieSearchResult
	s float64
}

type scoredActor struct {
	r *model.ActorSearchResult
	s float64
}

// yearTier: 1.0 when the release year equals the searched year, 0.5 when
// it's off by one, 0.0 otherwise. Returns 0 when either side is unknown.
func yearTier(wantYear int, got datatypes.Date) float64 {
	if wantYear == 0 {
		return 0
	}
	gotYear := time.Time(got).Year()
	if gotYear == 0 {
		return 0
	}
	switch diff := wantYear - gotYear; diff {
	case 0:
		return 1.0
	case 1, -1:
		return 0.5
	}
	return 0
}

// titleTier: Jaro–Winkler similarity (0..1) plus an exact-substring bonus
// capped at 1.0. "Fight Club" matching "Fight Club (1999)" returns ~1.0.
func titleTier(keyword, title string) float64 {
	if keyword == "" || title == "" {
		return 0
	}
	sim := comparer.CompareTitle(keyword, title)
	// Exact phrase bonus: keyword appears as a case-insensitive substring of title.
	if strings.Contains(strings.ToLower(title), strings.ToLower(keyword)) {
		sim += 0.1
	}
	if sim > 1.0 {
		sim = 1.0
	}
	return sim
}

// priorityTier normalises the provider priority into [0..1].
func priorityTier(provider string, lookup PriorityOf, maxPriority float64) float64 {
	if lookup == nil || maxPriority <= 0 {
		return 0
	}
	p := lookup(provider)
	if p <= 0 {
		return 0
	}
	if p >= maxPriority {
		return 1
	}
	return p / maxPriority
}

// languageTier returns a score in [0..1] reflecting how well a result's
// title matches the requested BCP 47 language in terms of Unicode script.
// When the user has no language preference it returns 0 (tier is disabled).
// When a result's primary script matches the expected script it returns 1,
// otherwise 0. Rationale: TMDB/TVDB/AniList already pick the translated
// title based on X-Is-Language, so a title whose script aligns with the
// request is almost always the localised version.
func languageTier(tag, title string) float64 {
	if tag == "" || title == "" {
		return 0
	}
	want := expectedScript(tag)
	if want == scriptAny {
		return 0
	}
	got := detectPrimaryScript(title)
	if got == want {
		return 1.0
	}
	return 0
}

// scriptClass labels the broad Unicode script a title is written in.
type scriptClass int

const (
	scriptAny scriptClass = iota
	scriptLatin
	scriptHan      // Chinese Hanzi (covers zh-CN/zh-TW/zh-HK)
	scriptJapanese // Hiragana/Katakana (Japanese-specific); Han-only is treated as Chinese
	scriptHangul
	scriptArabic
	scriptHebrew
	scriptThai
	scriptDevanagari
	scriptCyrillic
	scriptGreek
)

// expectedScript maps the primary subtag of a BCP 47 language tag to the
// Unicode script family its titles are usually written in.
func expectedScript(tag string) scriptClass {
	base := strings.ToLower(tag)
	if i := strings.IndexAny(base, "-_"); i > 0 {
		base = base[:i]
	}
	switch base {
	case "zh", "yue", "wuu":
		return scriptHan
	case "ja":
		return scriptJapanese
	case "ko":
		return scriptHangul
	case "ar", "fa", "ur":
		return scriptArabic
	case "he":
		return scriptHebrew
	case "th":
		return scriptThai
	case "hi", "bn", "mr", "ta", "te":
		return scriptDevanagari
	case "ru", "uk", "bg", "sr", "mk":
		return scriptCyrillic
	case "el":
		return scriptGreek
	case "en", "fr", "de", "es", "it", "pt", "nl", "sv", "da", "no",
		"fi", "cs", "pl", "hu", "ro", "sk", "sl", "hr", "et", "lv",
		"lt", "tr", "vi", "id", "ms":
		return scriptLatin
	}
	return scriptAny
}

// detectPrimaryScript counts the script membership of each letter rune in
// the title and returns the most common one. Non-letters, spaces and
// punctuation are ignored so "Spider-Man: No Way Home" still classifies as
// Latin and "搏击俱乐部 (1999)" still classifies as Han.
func detectPrimaryScript(title string) scriptClass {
	var counts [scriptGreek + 1]int
	for _, r := range title {
		if !unicode.IsLetter(r) {
			continue
		}
		// Japanese-first: kana unambiguously means Japanese.
		if unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			counts[scriptJapanese]++
			continue
		}
		switch {
		case unicode.Is(unicode.Han, r):
			counts[scriptHan]++
		case unicode.Is(unicode.Hangul, r):
			counts[scriptHangul]++
		case unicode.Is(unicode.Arabic, r):
			counts[scriptArabic]++
		case unicode.Is(unicode.Hebrew, r):
			counts[scriptHebrew]++
		case unicode.Is(unicode.Thai, r):
			counts[scriptThai]++
		case unicode.Is(unicode.Devanagari, r) ||
			unicode.Is(unicode.Bengali, r) ||
			unicode.Is(unicode.Tamil, r) ||
			unicode.Is(unicode.Telugu, r):
			counts[scriptDevanagari]++
		case unicode.Is(unicode.Cyrillic, r):
			counts[scriptCyrillic]++
		case unicode.Is(unicode.Greek, r):
			counts[scriptGreek]++
		case unicode.Is(unicode.Latin, r):
			counts[scriptLatin]++
		}
	}
	best := scriptAny
	bestN := 0
	for s := scriptLatin; s <= scriptGreek; s++ {
		if counts[s] > bestN {
			best = s
			bestN = counts[s]
		}
	}
	// Japanese titles often mix Han + kana; if any kana is present prefer
	// Japanese even when Han count is higher.
	if counts[scriptJapanese] > 0 && best == scriptHan {
		best = scriptJapanese
	}
	return best
}

// promoteTopPerProvider scans an already-sorted slice and lifts the first
// occurrence of each provider to the top (in the order they appear), so
// every provider gets a first-screen slot. The remaining elements keep
// their relative order below the promoted head.
func promoteTopPerProvider[T any](sorted []T, providerOf func(T) string) []T {
	if len(sorted) <= 1 {
		return sorted
	}
	seen := make(map[string]struct{})
	head := make([]T, 0, len(sorted))
	tail := make([]T, 0, len(sorted))
	for _, r := range sorted {
		p := providerOf(r)
		if _, dup := seen[p]; dup {
			tail = append(tail, r)
			continue
		}
		seen[p] = struct{}{}
		head = append(head, r)
	}
	return append(head, tail...)
}
