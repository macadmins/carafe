package brew

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-version"

	"github.com/macadmins/carafe/exec"
)

// ErrUsage marks an error in the arguments, not in Homebrew. Check returns the
// failing code for it like for any other error, so under --munki-installcheck
// a typo in --min-version makes the item fail visibly instead of reading as
// compliant. Callers can use it to show usage help.
var ErrUsage = errors.New("invalid arguments")

type CheckResult struct {
	Installed           bool   `json:"installed"`
	Version             string `json:"version"`
	MeetsMinimumVersion bool   `json:"meets_minimum_version"`
	Name                string `json:"name"`
}

// Check verifies whether item is installed and optionally meets minVersion.
// cacheTTL controls how long the brew info cache is valid; pass 0 to disable caching.
//
// When it returns an error, the exit code is always the failing one for the
// selected mode. With munkiInstallsCheck that is 0 ("install needed"), so a
// broken brew, an unreadable version or a bad argument is never reported as
// compliant. Argument errors wrap ErrUsage.
func Check(c exec.CarafeConfig, item, minVersion string, munkiInstallsCheck, skipNotInstalled bool, cacheTTL time.Duration) (int, error) {
	pass := 0
	fail := 1

	if munkiInstallsCheck {
		pass = 1
		fail = 0
	}

	if minVersion == "" && skipNotInstalled {
		return fail, fmt.Errorf("%w: --min-version must be set when --skip-not-installed is set", ErrUsage)
	}
	// Validate with go-version. The version comparison accepts every string
	// that go-version accepts, so a value that passes here does not fail later
	// as a runtime error. A revision such as "2.50.1_1" is rejected: it is not
	// a version that can be enforced.
	if minVersion != "" {
		if _, err := version.NewVersion(minVersion); err != nil {
			return fail, fmt.Errorf("%w: --min-version %q is not a valid version: %v", ErrUsage, minVersion, err)
		}
	}

	result := CheckResult{
		Name: item,
	}

	// make sure brew is installed - check the path is present
	_, err := c.CUSudo.OSFunc.Stat(c.GetBrewPath())
	if err != nil {
		if os.IsNotExist(err) {
			return pass, nil // if brew is not installed we do not need to do anything
		}
	}

	output, err := getInfoOutput(c, item, cacheTTL)
	if err != nil {
		// brew knows no formula with this name and the Cellar has no keg for
		// it, so it is not installed. With --skip-not-installed that gives the
		// same result as any formula that is not installed: pass (exit 1 in
		// Munki mode, 0 otherwise). A keg without a loadable formula (removed
		// from its tap, or the tap was untapped) is still installed, so that
		// case fails closed.
		if skipNotInstalled && isUnknownFormula(err, item) && !rackExists(c, item) {
			return pass, printResultJSON(result)
		}
		return fail, err
	}

	// is it installed?
	isPresent, err := installed(output)
	if err != nil {
		return fail, err
	}

	result.Installed = isPresent
	if !isPresent {
		err = printResultJSON(result)
		if err != nil {
			return fail, err
		}
		if skipNotInstalled {
			return pass, nil
		}
		return fail, nil
	}

	// get the version
	installedVersion, err := getVersion(output)
	if err != nil {
		return fail, err
	}

	result.Version = installedVersion

	if minVersion != "" {
		// Use the already-fetched output to avoid a second brew call.
		meetsMinimum, err := meetsMinimumFromOutput(output, item, minVersion)
		if err != nil {
			return fail, err
		}

		result.MeetsMinimumVersion = meetsMinimum
		if !meetsMinimum {
			err = printResultJSON(result)
			if err != nil {
				return fail, err
			}
			return fail, nil
		}
	}
	return pass, printResultJSON(result)
}

// getInfoOutput fetches brew info JSON for item, using the disk cache when cacheTTL > 0.
func getInfoOutput(c exec.CarafeConfig, item string, cacheTTL time.Duration) (string, error) {
	if cacheTTL > 0 {
		return infoOutputCached(c, item, cachePath(c.GetBrewPath()), cacheTTL)
	}
	return infoOutput(c, item)
}

func printResultJSON(result CheckResult) error {
	// Print the result as JSON
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(b))
	return nil
}

// isUnknownFormula reports whether a brew error says that no formula has the
// name item, for example after a typo, when homebrew-core removed it, or when
// its tap is not tapped. Only the requested name counts, so a missing
// dependency does not read as "not installed". brew prints the name in lower
// case, so the match ignores case. A formula file that exists but is broken
// ("In formula file: ...") is an error, not an unknown formula.
func isUnknownFormula(err error, item string) bool {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "in formula file:") {
		return false
	}
	return strings.Contains(msg, strings.ToLower(fmt.Sprintf("No available formula with the name %q", item)))
}

// rackExists reports whether the Cellar may hold a keg for item. Only a
// definite "does not exist" returns false; any other Stat error counts as
// present, so the caller fails closed.
func rackExists(c exec.CarafeConfig, item string) bool {
	prefix := filepath.Dir(filepath.Dir(c.GetBrewPath()))
	rack := filepath.Join(prefix, "Cellar", strings.ToLower(path.Base(item)))
	_, err := c.CUSudo.OSFunc.Stat(rack)
	return !os.IsNotExist(err)
}
