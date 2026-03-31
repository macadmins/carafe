package brew

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/macadmins/carafe/exec"
)

const (
	// cacheDir is owned by root (mode 0700) so non-root users cannot read,
	// write, or pre-create files inside it, preventing symlink/injection attacks.
	cacheDir        = "/var/root/.carafe"
	cacheFileArm64  = cacheDir + "/brew_info_cache_arm64.json"
	cacheFileX86_64 = cacheDir + "/brew_info_cache_x86_64.json"
)

// cachePath returns the cache file path for the given brew executable path.
func cachePath(brewPath string) string {
	if brewPath == "/opt/homebrew/bin/brew" {
		return cacheFileArm64
	}
	return cacheFileX86_64
}

// infoOutputCached is like infoOutput but uses a filesystem cache of
// `brew info --json --installed` to avoid calling brew once per formula.
// The cache at cacheFile is refreshed when it is older than ttl.
// On any cache error it falls back to a direct brew call.
// If the formula is not present in the installed cache it falls back to a
// direct brew call rather than synthesising a "not installed" response, so
// that typos and unresolved aliases are still caught by brew.
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

	// Formula not found in the installed list. Fall back to a direct brew call
	// so that typos or aliases are handled correctly (brew will error on an
	// unknown name rather than silently reporting it as not-installed).
	return infoOutput(c, item)
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

	// Best-effort write; ignore errors so that a missing or unwritable cache
	// directory never breaks the check — we just skip caching that run.
	if mkErr := os.MkdirAll(filepath.Dir(cacheFile), 0700); mkErr == nil {
		_ = os.WriteFile(cacheFile, []byte(out), 0600)
	}

	return formulas, nil
}
