package brew

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/macadmins/carafe/exec"
)

// cachePath returns the cache file path for the given brew executable path.
// Different paths are used for arm64 and x86_64 to avoid collisions.
func cachePath(brewPath string) string {
	if brewPath == "/opt/homebrew/bin/brew" {
		return "/tmp/carafe_brew_info_cache_arm64.json"
	}
	return "/tmp/carafe_brew_info_cache_x86_64.json"
}

// infoOutputCached is like infoOutput but uses a filesystem cache of
// `brew info --json --installed` to avoid calling brew once per formula.
// The cache at cacheFile is refreshed when it is older than ttl.
// On any cache error it falls back to a direct brew call.
func infoOutputCached(c exec.CarafeConfig, item, cacheFile string, ttl time.Duration) (string, error) {
	formulas, err := loadOrRefreshCache(c, cacheFile, ttl)
	if err != nil {
		return infoOutput(c, item)
	}

	for _, f := range formulas {
		if f.Name == item {
			b, jsonErr := json.Marshal([]HomebrewFormula{f})
			if jsonErr != nil {
				return infoOutput(c, item)
			}
			return string(b), nil
		}
	}

	// Not found in the installed list – formula is not installed.
	b, jsonErr := json.Marshal([]HomebrewFormula{{Name: item, Installed: []Installed{}}})
	if jsonErr != nil {
		return infoOutput(c, item)
	}
	return string(b), nil
}

// loadOrRefreshCache returns the list of installed Homebrew formulas, reading
// from cacheFile when it is younger than ttl, or running `brew info --json
// --installed` and writing a fresh cache otherwise.
func loadOrRefreshCache(c exec.CarafeConfig, cacheFile string, ttl time.Duration) ([]HomebrewFormula, error) {
	info, err := os.Stat(cacheFile)
	if err == nil && time.Since(info.ModTime()) < ttl {
		data, readErr := os.ReadFile(cacheFile)
		if readErr == nil {
			var formulas []HomebrewFormula
			if jsonErr := json.Unmarshal(data, &formulas); jsonErr == nil {
				return formulas, nil
			}
		}
	}

	out, err := c.RunBrew([]string{"info", "--json", "--installed"})
	if err != nil {
		return nil, fmt.Errorf("brew info --installed: %w", err)
	}

	var formulas []HomebrewFormula
	if err := json.Unmarshal([]byte(out), &formulas); err != nil {
		return nil, fmt.Errorf("parse brew info output: %w", err)
	}

	// Best-effort write; ignore errors so a read-only /tmp never breaks the check.
	_ = os.WriteFile(cacheFile, []byte(out), 0600)

	return formulas, nil
}
