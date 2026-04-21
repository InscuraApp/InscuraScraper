package comparer

import (
	"github.com/adrg/strutil/metrics"
)

// Compare returns the Levenshtein similarity between two strings (0..1),
// suitable for comparing short IDs / numbers where character-level edits
// matter.
func Compare(a, b string) float64 {
	m := &metrics.Levenshtein{
		CaseSensitive: false,
		InsertCost:    1,
		DeleteCost:    1,
		ReplaceCost:   2,
	}
	return m.Compare(a, b)
}

// CompareTitle returns the Jaro–Winkler similarity between two strings (0..1).
// It's better suited than Levenshtein for short title-like text because it
// rewards common prefixes and is length-normalised.
func CompareTitle(a, b string) float64 {
	m := metrics.NewJaroWinkler()
	m.CaseSensitive = false
	return m.Compare(a, b)
}
