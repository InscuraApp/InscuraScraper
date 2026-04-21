package anilist

import (
	"testing"

	"inscurascraper/provider/internal/testkit"
)

// AniList API is public, no authentication needed.

func TestAniList_GetMovieInfoByID(t *testing.T) {
	testkit.Test(t, New, []string{
		"anime:1",     // Cowboy Bebop
		"anime:16498", // Shingeki no Kyojin (Attack on Titan)
		"manga:30013", // One Piece (manga)
	})
}

func TestAniList_GetMovieInfoByURL(t *testing.T) {
	testkit.Test(t, New, []string{
		"https://anilist.co/anime/1/Cowboy-Bebop",
		"https://anilist.co/manga/30013/One-Piece",
	})
}

func TestAniList_SearchMovie(t *testing.T) {
	testkit.Test(t, New, []string{
		"Cowboy Bebop",
		"Attack on Titan",
	})
}

func TestAniList_GetActorInfoByID(t *testing.T) {
	testkit.Test(t, New, []string{
		"97009", // Shinichirou Watanabe
		"95011", // Kouichi Yamadera
	})
}

func TestAniList_SearchActor(t *testing.T) {
	testkit.Test(t, New, []string{
		"Watanabe",
	})
}
