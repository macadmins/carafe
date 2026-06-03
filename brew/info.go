package brew

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/hashicorp/go-version"

	"github.com/macadmins/carafe/exec"
)

var brewRevisionRe = regexp.MustCompile(`_\d+$`)

const DefaultInventoryPath = "/Library/Application Support/MacAdmins/Carafe/homebrew_formulae.json"

// stripBrewRevision removes the Homebrew revision suffix (e.g. "_1") from a version string.
// Homebrew appends _N to indicate a package revision, but this is not valid semver.
func stripBrewRevision(v string) string {
	return brewRevisionRe.ReplaceAllString(v, "")
}

type HomebrewFormula struct {
	Name               string      `json:"name"`
	FullName           string      `json:"full_name"`
	Tap                string      `json:"tap"`
	Homepage           string      `json:"homepage"`
	License            string      `json:"license"`
	URLs               URLs        `json:"urls"`
	Installed          []Installed `json:"installed"`
	Outdated           bool        `json:"outdated"`
	InstalledOnRequest bool        `json:"installed_on_request"`
}

type Installed struct {
	Version            string `json:"version"`
	InstalledOnRequest bool   `json:"installed_on_request"`
}

type URLs struct {
	Stable URLInfo `json:"stable"`
}

type URLInfo struct {
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
}

type BrewInfoV2 struct {
	Formulae []HomebrewFormula `json:"formulae"`
}

type FormulaInventory struct {
	SchemaVersion string                `json:"schema_version"`
	GeneratedAt   string                `json:"generated_at"`
	BrewPath      string                `json:"brew_path"`
	Arch          string                `json:"arch"`
	Formulae      []FormulaInventoryRow `json:"formulae"`
}

type FormulaInventoryRow struct {
	Name               string `json:"name"`
	FullName           string `json:"full_name"`
	Tap                string `json:"tap"`
	InstalledVersion   string `json:"installed_version"`
	InstalledOnRequest bool   `json:"installed_on_request"`
	SourceURL          string `json:"source_url"`
	Homepage           string `json:"homepage"`
	License            string `json:"license"`
	Checksum           string `json:"checksum"`
	Outdated           bool   `json:"outdated"`
}

func AllInfo(c exec.CarafeConfig) error {
	args := []string{"info", "--json", "--installed"}
	_, err := c.RunBrewWithOutput(args)
	return err
}

func FormulaInventoryOutput(c exec.CarafeConfig, generatedAt time.Time) (FormulaInventory, error) {
	out, err := c.RunBrew([]string{"info", "--json=v2", "--installed"})
	if err != nil {
		return FormulaInventory{}, err
	}

	return formulaInventoryFromOutput(out, c.GetBrewPath(), c.Arch, generatedAt)
}

func WriteFormulaInventory(c exec.CarafeConfig, path string) error {
	inventory, err := FormulaInventoryOutput(c, time.Now().UTC())
	if err != nil {
		return err
	}

	return WriteFormulaInventoryData(path, inventory)
}

func WriteFormulaInventoryData(path string, inventory FormulaInventory) error {
	data, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func PrintFormulaInventory(inventory FormulaInventory) error {
	data, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func formulaInventoryFromOutput(output, brewPath, arch string, generatedAt time.Time) (FormulaInventory, error) {
	var info BrewInfoV2
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		return FormulaInventory{}, err
	}

	inventory := FormulaInventory{
		SchemaVersion: "1",
		GeneratedAt:   generatedAt.UTC().Format(time.RFC3339),
		BrewPath:      brewPath,
		Arch:          arch,
		Formulae:      []FormulaInventoryRow{},
	}

	for _, formula := range info.Formulae {
		if len(formula.Installed) == 0 {
			continue
		}

		installed := formula.Installed[0]
		inventory.Formulae = append(inventory.Formulae, FormulaInventoryRow{
			Name:               formula.Name,
			FullName:           formula.FullName,
			Tap:                formula.Tap,
			InstalledVersion:   installed.Version,
			InstalledOnRequest: installed.InstalledOnRequest,
			SourceURL:          formula.URLs.Stable.URL,
			Homepage:           formula.Homepage,
			License:            formula.License,
			Checksum:           formula.URLs.Stable.Checksum,
			Outdated:           formula.Outdated,
		})
	}

	return inventory, nil
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
