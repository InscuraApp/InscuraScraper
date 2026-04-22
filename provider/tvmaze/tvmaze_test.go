package tvmaze

import (
	"inscurascraper/provider/internal/testkit"
	"os"
	"testing"
)

// TVMaze public API does not require authentication.
// Set env IS_TVMAZE_API_KEY if needed for premium features.
var apiKey = os.Getenv("IS_TVMAZE_API_KEY")

func TestTVMaze_GetMovieInfoByID(t *testing.T) {
	testkit.Test(t, func() *TVMaze {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"169", // Breaking Bad
		"82",  // Game of Thrones
	})
}

func TestTVMaze_GetMovieInfoByURL(t *testing.T) {
	testkit.Test(t, func() *TVMaze {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"https://www.tvmaze.com/shows/169/breaking-bad",
		"https://www.tvmaze.com/shows/82/game-of-thrones",
	})
}

func TestTVMaze_SearchMovie(t *testing.T) {
	testkit.Test(t, func() *TVMaze {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"Breaking Bad",
		"Game of Thrones",
	})
}

func TestTVMaze_GetActorInfoByID(t *testing.T) {
	testkit.Test(t, func() *TVMaze {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"14245", // Bryan Cranston
		"1",     // Mike Vogel
	})
}

func TestTVMaze_SearchActor(t *testing.T) {
	testkit.Test(t, func() *TVMaze {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"Bryan Cranston",
	})
}
