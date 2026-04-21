package envconfig

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvConfigs(t *testing.T) {
	// Set mock environment variables
	os.Clearenv()
	for _, unit := range []struct {
		key, value string
	}{
		{"IS_ACTOR_PROVIDER_ABC__PRIORITY", "1"},
		{"IS_MOVIE_PROVIDER_DEF__PRIORITY", "2"},
		{"IS_MOVIE_PROVIDER_xyz__PRIORITY", "0"},
		{"IS_MOVIE_PROVIDER_OPQ__TOKEN", "token1"},
		{"IS_PROVIDER_UVW__TIMEout", "900s"},
		{"IS_PROVIDER_UVW__tOkEn", "wrong"},
		{"IS_MOVIE_PROVIDER_UVW__tOkEn", "token2"},   // override movie
		{"IS_PROVIDER_UVW__PRIORITY", "5"},           // -> actor/movie
		{"IS_MOVIE_PROVIDER_UVW__PRIORITY", "0"},     // override movie
		{"IS_MOVIE_PROVIDER_JJJ_KKK__PRIORITY", "0"}, // hyphen in name
		{"irrelevant_key", "ignore_me"},
		{"mt_malformed_key", "ignore_me"},
	} {
		err := os.Setenv(unit.key, unit.value)
		require.NoError(t, err)
	}

	// Reload
	InitAllEnvConfigs()

	for k, v := range map[string]float64{
		"ABC": 1,
		"UVW": 5,
	} {
		priority, err := ActorProviderConfigs.GetOrDefault(k).GetFloat64("priority")
		if assert.NoError(t, err) {
			assert.Equal(t, v, priority)
		}
	}

	for k, v := range map[string]float64{
		"DEF":     2,
		"XYZ":     0,
		"UVW":     0,
		"JJJ_KKK": 0,
		"JJJ-KKK": 0,
	} {
		priority, err := MovieProviderConfigs.GetOrDefault(k).GetFloat64("priority")
		if assert.NoError(t, err) {
			assert.Equal(t, v, priority)
		}
	}

	val, err := MovieProviderConfigs.GetOrDefault("OPQ").GetString("token")
	if assert.NoError(t, err) {
		assert.Equal(t, "token1", val)
	}

	val, err = MovieProviderConfigs.GetOrDefault("UVW").GetString("ToKeN")
	if assert.NoError(t, err) {
		assert.Equal(t, "token2", val)
	}

	timeout, err := MovieProviderConfigs.GetOrDefault("UVW").GetDuration("timeout")
	if assert.NoError(t, err) {
		assert.Equal(t, 900*time.Second, timeout)
	}
}
