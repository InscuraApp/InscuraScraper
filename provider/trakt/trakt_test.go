package trakt

import (
	"inscurascraper/provider/internal/testkit"
	"os"
	"testing"
)

// Set env IS_TRAKT_CLIENT_ID to run tests.
var clientID = os.Getenv("IS_TRAKT_CLIENT_ID")

func TestTrakt_GetMovieInfoByID(t *testing.T) {
	if clientID == "" {
		t.Skip("IS_TRAKT_CLIENT_ID is not set")
	}
	testkit.Test(t, func() *Trakt {
		res := New()
		res.clientID = clientID
		return res
	}, []string{
		"fight-club",
		"the-godfather",
		"inception",
	})
}

func TestTrakt_GetMovieInfoByURL(t *testing.T) {
	if clientID == "" {
		t.Skip("IS_TRAKT_CLIENT_ID is not set")
	}
	testkit.Test(t, func() *Trakt {
		res := New()
		res.clientID = clientID
		return res
	}, []string{
		"https://trakt.tv/movies/fight-club",
		"https://trakt.tv/movies/the-godfather",
	})
}

func TestTrakt_SearchMovie(t *testing.T) {
	if clientID == "" {
		t.Skip("IS_TRAKT_CLIENT_ID is not set")
	}
	testkit.Test(t, func() *Trakt {
		res := New()
		res.clientID = clientID
		return res
	}, []string{
		"Fight Club",
		"Inception",
	})
}

func TestTrakt_GetActorInfoByID(t *testing.T) {
	if clientID == "" {
		t.Skip("IS_TRAKT_CLIENT_ID is not set")
	}
	testkit.Test(t, func() *Trakt {
		res := New()
		res.clientID = clientID
		return res
	}, []string{
		"brad-pitt",
		"leonardo-dicaprio",
	})
}

func TestTrakt_GetActorInfoByURL(t *testing.T) {
	if clientID == "" {
		t.Skip("IS_TRAKT_CLIENT_ID is not set")
	}
	testkit.Test(t, func() *Trakt {
		res := New()
		res.clientID = clientID
		return res
	}, []string{
		"https://trakt.tv/people/brad-pitt",
		"https://trakt.tv/people/leonardo-dicaprio",
	})
}

func TestTrakt_SearchActor(t *testing.T) {
	if clientID == "" {
		t.Skip("IS_TRAKT_CLIENT_ID is not set")
	}
	testkit.Test(t, func() *Trakt {
		res := New()
		res.clientID = clientID
		return res
	}, []string{
		"Brad Pitt",
		"Leonardo",
	})
}
