package brew

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/macadmins/carafe/exec"
)

var brewRevisionRe = regexp.MustCompile(`_\d+$`)

// stripBrewRevision removes the Homebrew revision suffix (e.g. "_1") from a version string.
// Homebrew appends _N to indicate a package revision, but this is not valid semver.
func stripBrewRevision(v string) string {
	return brewRevisionRe.ReplaceAllString(v, "")
}

type HomebrewFormula struct {
	Name      string      `json:"name"`
	Installed []Installed `json:"installed"`
	LinkedKeg string      `json:"linked_keg,omitempty"`
}

type Installed struct {
	Version string `json:"version"`
}

func AllInfo(c exec.CarafeConfig) error {
	args := []string{"info", "--json", "--installed"}
	_, err := c.RunBrewWithOutput(args)
	return err
}

func Info(c exec.CarafeConfig, item string) error {
	args := []string{"info", "--json", item}
	_, err := c.RunBrewWithOutput(args)
	return err
}

func infoOutput(c exec.CarafeConfig, item string) (string, error) {
	args := []string{"info", "--json", item}
	out, err := c.RunBrew(args)
	return out, err
}

func installed(output string) (bool, error) {
	var info []HomebrewFormula
	err := json.Unmarshal([]byte(output), &info)
	if err != nil {
		return false, err
	}

	if len(info) == 0 {
		return false, fmt.Errorf("empty JSON array")
	}

	if len(info[0].Installed) == 0 {
		return false, nil
	}
	return true, nil
}

func IsInstalled(c exec.CarafeConfig, item string) (bool, error) {
	out, err := infoOutput(c, item)
	if err != nil {
		return false, err
	}

	return installed(out)
}

func getVersion(output string) (string, error) {
	var info []HomebrewFormula
	err := json.Unmarshal([]byte(output), &info)
	if err != nil {
		return "", err
	}

	if len(info) == 0 || len(info[0].Installed) == 0 {
		return "", nil
	}
	// The linked keg is the version in use. Use it when it is a stable keg that
	// brew lists as installed. Otherwise (a keg-only formula, a linked HEAD keg,
	// or a link to a keg that is gone) use the newest stable keg: brew sorts
	// "installed" from oldest to newest, and an old keg stays there until
	// "brew cleanup" removes it. A HEAD version has no number to compare, so a
	// HEAD keg is used only when no stable keg exists.
	kegs := info[0].Installed
	linked := info[0].LinkedKeg
	if linked != "" && !isHeadVersion(linked) {
		for _, k := range kegs {
			if k.Version == linked {
				return linked, nil
			}
		}
	}
	for i := len(kegs) - 1; i >= 0; i-- {
		if !isHeadVersion(kegs[i].Version) {
			return kegs[i].Version, nil
		}
	}
	return kegs[len(kegs)-1].Version, nil
}

// isHeadVersion reports whether a keg version is a HEAD build, such as
// "HEAD-1234567".
func isHeadVersion(v string) bool {
	return strings.HasPrefix(v, "HEAD")
}

func InstalledVersion(c exec.CarafeConfig, item string) (string, error) {
	out, err := infoOutput(c, item)
	if err != nil {
		return "", err
	}

	return getVersion(out)
}

func VersionMeetsOrExceedsMinimum(c exec.CarafeConfig, item, minimumVersion string) (bool, error) {
	out, err := infoOutput(c, item)
	if err != nil { // couldn't get the state, return true to be safe
		return true, err
	}
	return meetsMinimumFromOutput(out, item, minimumVersion)
}

// meetsMinimumFromOutput performs the version comparison using already-fetched
// brew info JSON output, avoiding a second brew call.
func meetsMinimumFromOutput(output, item, minimumVersion string) (bool, error) {
	isInstalled, err := installed(output)
	if err != nil {
		return true, err
	}

	if !isInstalled {
		return true, nil // not installed, so it meets the minimum
	}

	installedVersion, err := getVersion(output)
	if err != nil {
		return true, err
	}

	if installedVersion == "" {
		return true, nil
	}

	parsedInstalledVersion, err := version.NewVersion(stripBrewRevision(installedVersion))
	if err != nil {
		return true, fmt.Errorf("failed to parse installed version %q for item %q: %w", installedVersion, item, err)
	}

	parsedMinimumVersion, err := version.NewVersion(minimumVersion)
	if err != nil {
		return true, fmt.Errorf("failed to parse minimum version %q for item %q: %w", minimumVersion, item, err)
	}

	return parsedInstalledVersion.GreaterThanOrEqual(parsedMinimumVersion), nil
}
