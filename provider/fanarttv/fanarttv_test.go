package fanarttv

import (
	"inscurascraper/provider/internal/testkit"
	"os"
	"testing"
)

// Set env IS_FANARTTV_API_KEY to run tests.
var apiKey = os.Getenv("IS_FANARTTV_API_KEY")

func TestFanartTV_GetMovieInfoByID(t *testing.T) {
	if apiKey == "" {
		t.Skip("IS_FANARTTV_API_KEY is not set")
	}
	testkit.Test(t, func() *FanartTV {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"movie:550", // Fight Club (TMDB ID)
		"movie:238", // The Godfather (TMDB ID)
		"tv:81189",  // Breaking Bad (TVDB ID)
		"tv:121361", // Game of Thrones (TVDB ID)
	})
}

func TestFanartTV_GetMovieInfoByURL(t *testing.T) {
	if apiKey == "" {
		t.Skip("IS_FANARTTV_API_KEY is not set")
	}
	testkit.Test(t, func() *FanartTV {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"https://fanart.tv/movie/550/fight-club/",
		"https://fanart.tv/series/81189/breaking-bad/",
	})
}
