package bangumi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"inscurascraper/provider/bangumi"
)

func TestBangumi_GetMovieInfoByID(t *testing.T) {
	b := bangumi.New()
	// 新世纪福音战士 (Neon Genesis Evangelion) subject ID 253
	info, err := b.GetMovieInfoByID("253")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "253", info.ID)
	assert.Equal(t, bangumi.Name, info.Provider)
	assert.NotEmpty(t, info.Title)
	assert.NotEmpty(t, info.Homepage)
	t.Logf("Title: %s, Score: %.1f, Genres: %v", info.Title, info.Score, info.Genres)
}

func TestBangumi_GetMovieInfoByURL(t *testing.T) {
	b := bangumi.New()
	info, err := b.GetMovieInfoByURL("https://bgm.tv/subject/253")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "253", info.ID)
}

func TestBangumi_SearchMovie(t *testing.T) {
	b := bangumi.New()
	results, err := b.SearchMovie("进击的巨人")
	require.NoError(t, err)
	assert.NotEmpty(t, results)
	t.Logf("Found %d results, first: %s (%.1f)", len(results), results[0].Title, results[0].Score)
}

func TestBangumi_GetActorInfoByID(t *testing.T) {
	b := bangumi.New()
	// 神谷浩史 (Hiroshi Kamiya) - person ID 3
	info, err := b.GetActorInfoByID("3")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "3", info.ID)
	assert.Equal(t, bangumi.Name, info.Provider)
	assert.NotEmpty(t, info.Name)
	assert.NotEmpty(t, info.Homepage)
	t.Logf("Name: %s, BloodType: %s, Nationality: %s", info.Name, info.BloodType, info.Nationality)
}

func TestBangumi_GetActorInfoByURL(t *testing.T) {
	b := bangumi.New()
	info, err := b.GetActorInfoByURL("https://bgm.tv/person/3")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "3", info.ID)
}

func TestBangumi_SearchActor(t *testing.T) {
	b := bangumi.New()
	results, err := b.SearchActor("神谷浩史")
	require.NoError(t, err)
	assert.NotEmpty(t, results)
	t.Logf("Found %d results, first: %s", len(results), results[0].Name)
}

func TestBangumi_NormalizeMovieID(t *testing.T) {
	b := bangumi.New()
	assert.Equal(t, "40028", b.NormalizeMovieID("40028"))
	assert.Equal(t, "", b.NormalizeMovieID("invalid"))
	assert.Equal(t, "", b.NormalizeMovieID(""))
}

func TestBangumi_ParseMovieIDFromURL(t *testing.T) {
	b := bangumi.New()

	id, err := b.ParseMovieIDFromURL("https://bgm.tv/subject/40028")
	assert.NoError(t, err)
	assert.Equal(t, "40028", id)

	_, err = b.ParseMovieIDFromURL("https://bgm.tv/person/3")
	assert.Error(t, err)

	_, err = b.ParseMovieIDFromURL("https://example.com/")
	assert.Error(t, err)
}

func TestBangumi_ParseActorIDFromURL(t *testing.T) {
	b := bangumi.New()

	id, err := b.ParseActorIDFromURL("https://bgm.tv/person/3")
	assert.NoError(t, err)
	assert.Equal(t, "3", id)

	_, err = b.ParseActorIDFromURL("https://bgm.tv/subject/40028")
	assert.Error(t, err)
}
