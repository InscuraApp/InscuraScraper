package tvmaze

import (
	"inscurascraper/provider/internal/testkit"
	"testing"
)

func TestTVMaze_GetMovieInfoByID(t *testing.T) {
	testkit.Test(t, New, []string{
		"169", // Breaking Bad
		"82",  // Game of Thrones
	})
}

func TestTVMaze_GetMovieInfoByURL(t *testing.T) {
	testkit.Test(t, New, []string{
		"https://www.tvmaze.com/shows/169/breaking-bad",
		"https://www.tvmaze.com/shows/82/game-of-thrones",
	})
}

func TestTVMaze_SearchMovie(t *testing.T) {
	testkit.Test(t, New, []string{
		"Breaking Bad",
		"Game of Thrones",
	})
}

func TestTVMaze_GetActorInfoByID(t *testing.T) {
	testkit.Test(t, New, []string{
		"14245", // Bryan Cranston
		"1",     // Mike Vogel
	})
}

func TestTVMaze_SearchActor(t *testing.T) {
	testkit.Test(t, New, []string{
		"Bryan Cranston",
	})
}
