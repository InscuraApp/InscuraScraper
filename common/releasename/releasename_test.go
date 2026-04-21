package releasename

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in        string
		wantTitle string
		wantYear  int
	}{
		{
			in:        "The.Adventures.of.Tintin.1991-1992.BluRay.1080p.AVC.LPCM.2.0",
			wantTitle: "The Adventures of Tintin",
			wantYear:  1992,
		},
		{
			in:        "Spider-Man.No.Way.Home.2021.1080p.BluRay.x265-RARBG",
			wantTitle: "Spider-Man No Way Home",
			wantYear:  2021,
		},
		{
			in:        "blade.runner.2049.2017.2160p.uhd.bluray.x265-terminal.mkv",
			wantTitle: "blade runner 2049", // PTN recognises 2049 as part of the title
			wantYear:  2017,
		},
		{
			in:        "The.Wolverine.2013.BluRay.1080p",
			wantTitle: "The Wolverine",
			wantYear:  2013,
		},
		{
			// No release markers — behaves as pass-through.
			in:        "Fight Club",
			wantTitle: "Fight Club",
			wantYear:  0,
		},
		{
			in:        "Avatar: Fire and Ash",
			wantTitle: "Avatar: Fire and Ash",
			wantYear:  0,
		},
		{
			// PTN truncates at the first dash; reclaim recovers the full title.
			in:        "Spider-Man.No.Way.Home.2021",
			wantTitle: "Spider-Man No Way Home",
			wantYear:  2021,
		},
		{
			// Bare year token at the end gets promoted.
			in:        "Some.Movie.2019",
			wantTitle: "Some Movie",
			wantYear:  2019,
		},
		{
			in:        "",
			wantTitle: "",
			wantYear:  0,
		},
	}

	for _, tc := range cases {
		got := Parse(tc.in)
		if got.Title != tc.wantTitle || got.Year != tc.wantYear {
			t.Errorf("Parse(%q) = {%q, %d}, want {%q, %d}",
				tc.in, got.Title, got.Year, tc.wantTitle, tc.wantYear)
		}
	}
}
