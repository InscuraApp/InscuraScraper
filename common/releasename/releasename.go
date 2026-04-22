// Package releasename extracts movie titles (and release years) from noisy
// release-style filenames like:
//
//	The.Adventures.of.Tintin.1991-1992.BluRay.1080p.AVC.LPCM.2.0
//	Spider-Man.No.Way.Home.2021.1080p.BluRay.x265-RARBG
//
// Under the hood it delegates to github.com/middelink/go-parse-torrent-name
// (a Go port of PTN) and layers small post-processing fixes on top:
//
//   - strip trailing 4-digit leftovers in the title (e.g. "Tintin 1991"
//     when PTN picked the second year of a range "1991-1992" as the year);
//   - collapse runs of whitespace / dots / underscores left over from the
//     filename so that providers receive a clean human-readable title.
//
// The package never fails: when the input has no obvious release markers
// (e.g. "Fight Club", "Avatar: Fire and Ash") it returns the input title
// with whitespace trimmed, and Year=0.
package releasename

import (
	"strings"
	"unicode"

	ptn "github.com/middelink/go-parse-torrent-name"
)

// Parsed is the output of Parse. Title is always populated (empty only if
// the input is whitespace). Year is 0 when no year could be extracted.
type Parsed struct {
	Title string
	Year  int
}

// Parse extracts the movie/series title and release year from a noisy
// filename or release name. See package doc for the post-processing applied
// on top of PTN.
func Parse(raw string) Parsed {
	in := strings.TrimSpace(raw)
	if in == "" {
		return Parsed{}
	}

	info, err := ptn.Parse(in)
	if err != nil || info == nil {
		return Parsed{Title: cleanTitle(in)}
	}

	title := cleanTitle(info.Title)
	year := info.Year

	// PTN sometimes truncates the title at a dash before it recognises a
	// later year boundary — e.g. "Spider-Man.No.Way.Home.2021" becomes
	// Title="Spider", Year=2021, dropping "Man No Way Home". Reclaim the
	// full title by splitting the raw input at the year token and taking
	// everything before it when the reclaimed title both extends and is
	// consistent with PTN's shorter guess.
	if year > 0 {
		if reclaimed, ok := reclaimTitleBeforeYear(in, year); ok && len(reclaimed) > len(title) {
			// Only accept the longer version if PTN's title is a prefix of
			// it (case-insensitive) — prevents over-grabbing when PTN's
			// detection was already better than our heuristic.
			if strings.HasPrefix(strings.ToLower(reclaimed), strings.ToLower(title)) {
				// Strip any residual trailing year (from ranges like
				// "1991-1992") so we don't keep "… Tintin 1991" behind.
				if cleaned, _ := stripTrailingYear(reclaimed, year); cleaned != "" {
					reclaimed = cleaned
				}
				title = reclaimed
			}
		}
	}

	// Post-process: when PTN picks the trailing year of a range like
	// "1991-1992" as Year, the Title often still ends with "... 1991".
	// If the title ends in a 4-digit number close to the parsed year,
	// drop it.
	if year > 0 {
		if t, stripped := stripTrailingYear(title, year); stripped {
			title = t
		}
	} else {
		// No year extracted — but the title may still end with a standalone
		// 19xx/20xx token that's clearly a year the parser missed; promote
		// it to Year so downstream providers can filter by it.
		if t, y, ok := promoteTrailingYear(title); ok {
			title = t
			year = y
		}
	}

	if title == "" {
		title = cleanTitle(in)
	}
	return Parsed{Title: title, Year: year}
}

// reclaimTitleBeforeYear splits the raw filename at the given year token
// and returns the cleaned text before it. Only matches a year that stands
// on its own (surrounded by '.', '_', '-' or whitespace, or at bounds) so
// we don't accidentally chop inside a title like "2049".
func reclaimTitleBeforeYear(raw string, year int) (string, bool) {
	needle := itoa4(year)
	// Scan for an isolated occurrence of the 4-digit year.
	i := 0
	for i < len(raw) {
		idx := strings.Index(raw[i:], needle)
		if idx < 0 {
			return "", false
		}
		start := i + idx
		end := start + 4
		if isBoundary(raw, start-1) && isBoundary(raw, end) {
			before := raw[:start]
			// Trim any trailing boundary character we're splitting on so
			// cleanTitle doesn't leave a dangling '-' (from year ranges
			// like "1991-1992", where a preceding "1991-" sits right in
			// front of our year boundary).
			before = strings.TrimRight(before, ".-_ \t")
			return cleanTitle(before), true
		}
		i = end
	}
	return "", false
}

func isBoundary(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	switch s[i] {
	case '.', '_', '-', ' ', '\t':
		return true
	}
	return false
}

func itoa4(n int) string {
	if n < 0 {
		n = -n
	}
	var b [4]byte
	for i := 3; i >= 0; i-- {
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[:])
}

func cleanTitle(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := true
	for _, r := range s {
		switch r {
		case '.', '_':
			r = ' '
		}
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}

// stripTrailingYear removes a trailing 4-digit number from the title when
// it differs from parsed year by at most 5 (covers "1991-1992" ranges and
// re-release dates like "Director's Cut 1990 1991"). Returns the cleaned
// title and whether a change was made.
func stripTrailingYear(title string, year int) (string, bool) {
	fields := strings.Fields(title)
	if len(fields) < 2 {
		return title, false
	}
	last := fields[len(fields)-1]
	if len(last) != 4 {
		return title, false
	}
	y := 0
	for _, r := range last {
		if r < '0' || r > '9' {
			return title, false
		}
		y = y*10 + int(r-'0')
	}
	if y < 1900 || y > 2100 {
		return title, false
	}
	diff := y - year
	if diff < 0 {
		diff = -diff
	}
	if diff > 5 {
		return title, false
	}
	return strings.Join(fields[:len(fields)-1], " "), true
}

// promoteTrailingYear detects a trailing 4-digit year in the title when
// PTN did not set Year. Returns (title without year, year, true) on success.
func promoteTrailingYear(title string) (string, int, bool) {
	fields := strings.Fields(title)
	if len(fields) < 2 {
		return title, 0, false
	}
	last := fields[len(fields)-1]
	if len(last) != 4 {
		return title, 0, false
	}
	y := 0
	for _, r := range last {
		if r < '0' || r > '9' {
			return title, 0, false
		}
		y = y*10 + int(r-'0')
	}
	if y < 1900 || y > 2100 {
		return title, 0, false
	}
	return strings.Join(fields[:len(fields)-1], " "), y, true
}
