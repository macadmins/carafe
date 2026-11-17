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
	return info[0].Installed[0].Version, nil
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

	meets, err := versionAtLeast(installedVersion, minimumVersion)
	if err != nil {
		return true, fmt.Errorf("failed to compare versions for item %q: %w", item, err)
	}
	return meets, nil
}

// brewSuffixRe matches a version with a letter suffix, such as "10.2p1"
// (openssh), "1.1.1w" (openssl@1.1), "2025b" (tzdb) or "3.0.0rc1". An optional
// leading "v" and an optional "-" before the letters are accepted, so
// "v10.2p2" and "1.0-rc1" take the same path as "10.2p2" and "1.0rc1". Group 1
// is the numeric part, group 2 the letters and group 3 an optional number
// after them.
var brewSuffixRe = regexp.MustCompile(`^v?(\d+(?:\.\d+)*)-?([a-z]+)(\d*)$`)

// suffixKind orders the kinds of suffix for one base version.
type suffixKind int

const (
	suffixPre  suffixKind = -1 // pre-release, such as rc1
	suffixNone suffixKind = 0  // no letter suffix
	suffixPost suffixKind = 1  // later release, such as p1
)

// preReleaseRanks orders the pre-release suffixes the way Homebrew does:
// alpha < beta < preview < pre < rc. "a" and "b" mean alpha and beta, but only
// with a number ("1.0a1"); see preReleaseRank.
var preReleaseRanks = map[string]int{ //nolint:gochecknoglobals
	"alpha": 1, "beta": 2, "preview": 3, "pre": 4, "rc": 5,
}

// brewVersion is a Homebrew version split into a base version and an
// optional letter suffix.
type brewVersion struct {
	base    *version.Version
	kind    suffixKind
	rank    int    // order of a pre-release suffix, see preReleaseRanks
	letters string // letters of a post-release suffix
	number  string // digits after the letters, leading zeros removed
}

// preReleaseRank returns the rank of a pre-release suffix, or false when the
// suffix marks a later release. It follows Homebrew's Version: alpha, beta,
// preview, pre, rc, and "a" or "b" with a number are pre-releases. Every other
// suffix is a later release, for example "10.2p1" (openssh), "1.1.1w"
// (openssl@1.1), "2025b" (tzdb), "4.5ga6" or "1.0dev1".
func preReleaseRank(letters, number string) (int, bool) {
	if rank, ok := preReleaseRanks[letters]; ok {
		return rank, true
	}
	if number != "" {
		switch letters {
		case "a":
			return preReleaseRanks["alpha"], true
		case "b":
			return preReleaseRanks["beta"], true
		}
	}
	return 0, false
}

// parseBrewVersion parses a Homebrew version without a revision suffix.
// go-version reads "10.2p1" as the pre-release "10.2-p1", which sorts before
// "10.2", and compares "rc10" before "rc9". So a letter suffix is kept apart
// from the base and compared the way Homebrew does. Other forms, such as
// upper-case suffixes or "+metadata", go to go-version unchanged. Every string
// that go-version accepts is accepted here too, and no other string is.
func parseBrewVersion(v string) (brewVersion, error) {
	if m := brewSuffixRe.FindStringSubmatch(v); m != nil {
		base, err := version.NewVersion(m[1])
		if err != nil {
			return brewVersion{}, err
		}
		bv := brewVersion{base: base, number: strings.TrimLeft(m[3], "0")}
		if rank, ok := preReleaseRank(m[2], m[3]); ok {
			bv.kind, bv.rank = suffixPre, rank
		} else {
			bv.kind, bv.letters = suffixPost, m[2]
		}
		return bv, nil
	}

	base, err := version.NewVersion(v)
	if err != nil {
		return brewVersion{}, err
	}
	return brewVersion{base: base, kind: suffixNone}, nil
}

// compareDigits compares two digit strings without leading zeros as numbers,
// with no limit on their length.
func compareDigits(a, b string) int {
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

// versionAtLeast reports whether the installed Homebrew version is at or
// above the minimum. Order for one base: "1.0a1" < "1.0b2" < "1.0rc10" <
// "1.0" < "1.0p1" < "1.0p2".
// The revision ("_1") of the installed version is ignored. A minimum with a
// revision is an error, as before: carafe cannot enforce a revision.
func versionAtLeast(installed, minimum string) (bool, error) {
	a, err := parseBrewVersion(stripBrewRevision(installed))
	if err != nil {
		return false, fmt.Errorf("installed version %q: %w", installed, err)
	}
	b, err := parseBrewVersion(minimum)
	if err != nil {
		return false, fmt.Errorf("minimum version %q: %w", minimum, err)
	}

	if c := a.base.Compare(b.base); c != 0 {
		return c > 0, nil
	}
	if a.kind != b.kind {
		return a.kind > b.kind, nil
	}
	if a.rank != b.rank {
		return a.rank > b.rank, nil
	}
	if a.letters != b.letters {
		return a.letters > b.letters, nil
	}
	return compareDigits(a.number, b.number) >= 0, nil
}
