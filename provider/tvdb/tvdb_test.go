package tvdb

import (
	"inscurascraper/provider/internal/testkit"
	"os"
	"testing"
)

// Set env IS_TVDB_API_KEY to run tests.
var apiKey = os.Getenv("IS_TVDB_API_KEY")

func TestTVDB_GetMovieInfoByID(t *testing.T) {
	if apiKey == "" {
		t.Skip("IS_TVDB_API_KEY is not set")
	}
	testkit.Test(t, func() *TVDB {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"series:81189",  // Breaking Bad
		"series:121361", // Game of Thrones
		"movie:12180",   // Inception
	})
}

func TestTVDB_GetMovieInfoByURL(t *testing.T) {
	if apiKey == "" {
		t.Skip("IS_TVDB_API_KEY is not set")
	}
	testkit.Test(t, func() *TVDB {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"https://www.thetvdb.com/series/breaking-bad",
		"https://www.thetvdb.com/movies/inception",
	})
}

func TestTVDB_SearchMovie(t *testing.T) {
	if apiKey == "" {
		t.Skip("IS_TVDB_API_KEY is not set")
	}
	testkit.Test(t, func() *TVDB {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"Breaking Bad",
		"Game of Thrones",
	})
}

func TestTVDB_GetActorInfoByID(t *testing.T) {
	if apiKey == "" {
		t.Skip("IS_TVDB_API_KEY is not set")
	}
	testkit.Test(t, func() *TVDB {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"255211", // Bryan Cranston
	})
}

func TestTVDB_SearchActor(t *testing.T) {
	if apiKey == "" {
		t.Skip("IS_TVDB_API_KEY is not set")
	}
	testkit.Test(t, func() *TVDB {
		res := New()
		res.apiKey = apiKey
		return res
	}, []string{
		"Bryan Cranston",
	})
}
