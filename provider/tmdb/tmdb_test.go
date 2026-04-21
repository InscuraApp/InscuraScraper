package tmdb

import (
	"os"
	"testing"

	"inscurascraper/provider/internal/testkit"
)

// Set env IS_TMDB_API_TOKEN to run tests.
var apiToken = os.Getenv("IS_TMDB_API_TOKEN")

func TestTMDB_GetMovieInfoByID(t *testing.T) {
	if apiToken == "" {
		t.Skip("IS_TMDB_API_TOKEN is not set")
	}
	testkit.Test(t, func() *TMDB {
		res := New()
		res.apiToken = apiToken
		return res
	}, []string{
		"550",   // Fight Club
		"238",   // The Godfather
		"27205", // Inception
	})
}

func TestTMDB_GetMovieInfoByURL(t *testing.T) {
	if apiToken == "" {
		t.Skip("IS_TMDB_API_TOKEN is not set")
	}
	testkit.Test(t, func() *TMDB {
		res := New()
		res.apiToken = apiToken
		return res
	}, []string{
		"https://www.themoviedb.org/movie/550-fight-club",
		"https://www.themoviedb.org/movie/238-the-godfather",
	})
}

func TestTMDB_SearchMovie(t *testing.T) {
	if apiToken == "" {
		t.Skip("IS_TMDB_API_TOKEN is not set")
	}
	testkit.Test(t, func() *TMDB {
		res := New()
		res.apiToken = apiToken
		return res
	}, []string{
		"Fight Club",
		"Inception",
	})
}

func TestTMDB_GetActorInfoByID(t *testing.T) {
	if apiToken == "" {
		t.Skip("IS_TMDB_API_TOKEN is not set")
	}
	testkit.Test(t, func() *TMDB {
		res := New()
		res.apiToken = apiToken
		return res
	}, []string{
		"287",   // Brad Pitt
		"6193",  // Leonardo DiCaprio
	})
}

func TestTMDB_GetActorInfoByURL(t *testing.T) {
	if apiToken == "" {
		t.Skip("IS_TMDB_API_TOKEN is not set")
	}
	testkit.Test(t, func() *TMDB {
		res := New()
		res.apiToken = apiToken
		return res
	}, []string{
		"https://www.themoviedb.org/person/287-brad-pitt",
		"https://www.themoviedb.org/person/6193-leonardo-dicaprio",
	})
}

func TestTMDB_SearchActor(t *testing.T) {
	if apiToken == "" {
		t.Skip("IS_TMDB_API_TOKEN is not set")
	}
	testkit.Test(t, func() *TMDB {
		res := New()
		res.apiToken = apiToken
		return res
	}, []string{
		"Brad Pitt",
		"Leonardo",
	})
}
