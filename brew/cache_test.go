package brew

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/macadmins/carafe/cudo"
	"github.com/macadmins/carafe/exec"
	"github.com/macadmins/carafe/shell/testshell"
)

func newTestConfig(outputs ...string) exec.CarafeConfig {
	return exec.CarafeConfig{
		Arch: "arm64",
		CUSudo: &cudo.CUSudo{
			CurrentUser: "testuser",
			Platform:    "darwin",
			OSFunc:      &cudo.MockOSFunc{},
			UserHome:    "/Users/testuser",
			Executor:    testshell.OutputExecutor(outputs...),
		},
	}
}

func TestLoadOrRefreshCache_CacheMiss(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cache.json")
	c := newTestConfig(TestInfoAllOutput)

	formulas, err := loadOrRefreshCache(c, cacheFile, 60*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, formulas)

	// Cache file should have been written
	_, statErr := os.Stat(cacheFile)
	assert.NoError(t, statErr)
}

func TestLoadOrRefreshCache_CacheHit(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cache.json")

	// Write a fresh cache file manually
	cachedFormulas := []HomebrewFormula{{Name: "htop", Installed: []Installed{{Version: "3.3.0"}}}}
	data, _ := json.Marshal(cachedFormulas)
	require.NoError(t, os.WriteFile(cacheFile, data, 0600))

	// Executor should NOT be called because the cache is fresh
	c := newTestConfig() // no outputs configured – would error if brew were called

	formulas, err := loadOrRefreshCache(c, cacheFile, 60*time.Second)
	require.NoError(t, err)
	require.Len(t, formulas, 1)
	assert.Equal(t, "htop", formulas[0].Name)
}

func TestLoadOrRefreshCache_CacheStale(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cache.json")

	// Write a cache file and backdate its mtime to make it stale
	staleFormulas := []HomebrewFormula{{Name: "old", Installed: []Installed{{Version: "0.1"}}}}
	data, _ := json.Marshal(staleFormulas)
	require.NoError(t, os.WriteFile(cacheFile, data, 0600))
	past := time.Now().Add(-2 * time.Minute)
	require.NoError(t, os.Chtimes(cacheFile, past, past))

	c := newTestConfig(TestInfoAllOutput)

	formulas, err := loadOrRefreshCache(c, cacheFile, 60*time.Second)
	require.NoError(t, err)
	// Should have refreshed; the stale "old" formula should not be returned
	for _, f := range formulas {
		assert.NotEqual(t, "old", f.Name)
	}
}

func TestLoadOrRefreshCache_BrewError(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cache.json")
	c := exec.CarafeConfig{
		Arch: "arm64",
		CUSudo: &cudo.CUSudo{
			CurrentUser: "testuser",
			Platform:    "darwin",
			OSFunc:      &cudo.MockOSFunc{},
			UserHome:    "/Users/testuser",
			Executor:    testshell.NewExecutor(testshell.AlwaysError(fmt.Errorf("brew failed"))),
		},
	}

	_, err := loadOrRefreshCache(c, cacheFile, 60*time.Second)
	assert.Error(t, err)
}

func TestInfoOutputCached_FormulaFound(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cache.json")

	// Pre-populate cache with htop installed
	cachedFormulas := []HomebrewFormula{{Name: "htop", Installed: []Installed{{Version: "3.3.0"}}}}
	data, _ := json.Marshal(cachedFormulas)
	require.NoError(t, os.WriteFile(cacheFile, data, 0600))

	c := newTestConfig() // brew should not be called

	out, err := infoOutputCached(c, "htop", cacheFile, 60*time.Second)
	require.NoError(t, err)

	var result []HomebrewFormula
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Len(t, result, 1)
	assert.Equal(t, "htop", result[0].Name)
	require.Len(t, result[0].Installed, 1)
	assert.Equal(t, "3.3.0", result[0].Installed[0].Version)
}

func TestInfoOutputCached_FormulaNotInstalled(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "cache.json")

	// Cache has no entries for wget
	cachedFormulas := []HomebrewFormula{{Name: "htop", Installed: []Installed{{Version: "3.3.0"}}}}
	data, _ := json.Marshal(cachedFormulas)
	require.NoError(t, os.WriteFile(cacheFile, data, 0600))

	c := newTestConfig()

	out, err := infoOutputCached(c, "wget", cacheFile, 60*time.Second)
	require.NoError(t, err)

	var result []HomebrewFormula
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	require.Len(t, result, 1)
	assert.Equal(t, "wget", result[0].Name)
	assert.Empty(t, result[0].Installed)
}

func TestCachePath(t *testing.T) {
	assert.Equal(t, "/tmp/carafe_brew_info_cache_arm64.json", cachePath("/opt/homebrew/bin/brew"))
	assert.Equal(t, "/tmp/carafe_brew_info_cache_x86_64.json", cachePath("/usr/local/bin/brew"))
}
