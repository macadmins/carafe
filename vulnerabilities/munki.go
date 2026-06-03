package vulnerabilities

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const DefaultOSVBatchURL = "https://api.osv.dev/v1/querybatch"

type MunkiPkginfoOptions struct {
	InventoryPaths []string
	MappingPath    string
	OutputDir      string
	Catalogs       []string
	CarafePath     string
	OSVURL         string
	HTTPClient     *http.Client
}

type MunkiPkginfoResult struct {
	PkginfoPaths                []string
	UnknownFormulae             []FormulaInventoryRow
	UnremediableVulnerabilities []UnremediableVulnerability
}

type UnremediableVulnerability struct {
	Formula          string   `json:"formula"`
	InstalledVersion string   `json:"installed_version"`
	VulnerabilityIDs []string `json:"vulnerability_ids"`
}

type FormulaInventory struct {
	Formulae []FormulaInventoryRow `json:"formulae"`
}

type FormulaInventoryRow struct {
	Name             string `json:"name"`
	FullName         string `json:"full_name"`
	InstalledVersion string `json:"installed_version"`
}

type OSVMapping struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}

type osvBatchRequest struct {
	Queries []osvQuery `json:"queries"`
}

type osvQuery struct {
	Version string     `json:"version"`
	Package osvPackage `json:"package"`
}

type osvPackage struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}

type osvBatchResponse struct {
	Results []osvResult `json:"results"`
}

type osvResult struct {
	Vulns []osvVulnerability `json:"vulns"`
}

type osvVulnerability struct {
	ID       string        `json:"id"`
	Aliases  []string      `json:"aliases"`
	Affected []osvAffected `json:"affected"`
}

type osvAffected struct {
	Package osvPackage `json:"package"`
	Ranges  []osvRange `json:"ranges"`
}

type osvRange struct {
	Events []osvEvent `json:"events"`
}

type osvEvent struct {
	Fixed string `json:"fixed"`
}

type mappedFormula struct {
	Formula FormulaInventoryRow
	Mapping OSVMapping
}

func GenerateMunkiPkginfos(opts MunkiPkginfoOptions) (MunkiPkginfoResult, error) {
	if len(opts.InventoryPaths) == 0 {
		return MunkiPkginfoResult{}, fmt.Errorf("--inventory must be set")
	}
	if opts.MappingPath == "" {
		return MunkiPkginfoResult{}, fmt.Errorf("--mapping must be set")
	}
	if opts.OutputDir == "" {
		return MunkiPkginfoResult{}, fmt.Errorf("--output-dir must be set")
	}
	if opts.CarafePath == "" {
		opts.CarafePath = "/opt/macadmins/bin/carafe"
	}
	if opts.OSVURL == "" {
		opts.OSVURL = DefaultOSVBatchURL
	}
	if len(opts.Catalogs) == 0 {
		opts.Catalogs = []string{"testing"}
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}

	mapping, err := loadMapping(opts.MappingPath)
	if err != nil {
		return MunkiPkginfoResult{}, err
	}
	formulae, err := loadInventoryFormulae(opts.InventoryPaths)
	if err != nil {
		return MunkiPkginfoResult{}, err
	}
	formulae = dedupeFormulae(formulae)

	var result MunkiPkginfoResult
	var queries []osvQuery
	var mapped []mappedFormula
	for _, formula := range formulae {
		identity, ok := mapping[formula.Name]
		if !ok || identity.Ecosystem == "" || identity.Name == "" {
			result.UnknownFormulae = append(result.UnknownFormulae, formula)
			continue
		}
		queries = append(queries, osvQuery{
			Version: formula.InstalledVersion,
			Package: osvPackage{
				Ecosystem: identity.Ecosystem,
				Name:      identity.Name,
			},
		})
		mapped = append(mapped, mappedFormula{Formula: formula, Mapping: identity})
	}

	osvResults, err := queryOSV(opts.HTTPClient, opts.OSVURL, queries)
	if err != nil {
		return MunkiPkginfoResult{}, err
	}
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return MunkiPkginfoResult{}, err
	}

	for i, osvResult := range osvResults {
		if len(osvResult.Vulns) == 0 {
			continue
		}
		fixedVersion := fixedVersion(osvResult.Vulns, mapped[i].Mapping)
		if fixedVersion == "" {
			result.UnremediableVulnerabilities = append(result.UnremediableVulnerabilities, UnremediableVulnerability{
				Formula:          mapped[i].Formula.Name,
				InstalledVersion: mapped[i].Formula.InstalledVersion,
				VulnerabilityIDs: vulnerabilityIDs(osvResult.Vulns),
			})
			continue
		}
		path, err := writeMunkiPkginfo(opts, mapped[i], osvResult.Vulns, fixedVersion)
		if err != nil {
			return MunkiPkginfoResult{}, err
		}
		result.PkginfoPaths = append(result.PkginfoPaths, path)
	}

	printSummary(result)
	return result, nil
}

func loadMapping(path string) (map[string]OSVMapping, error) {
	var mapping map[string]OSVMapping
	if err := readJSON(path, &mapping); err != nil {
		return nil, fmt.Errorf("read mapping: %w", err)
	}
	return mapping, nil
}

func loadInventoryFormulae(paths []string) ([]FormulaInventoryRow, error) {
	var formulae []FormulaInventoryRow
	for _, path := range paths {
		var inventory FormulaInventory
		if err := readJSON(path, &inventory); err != nil {
			return nil, fmt.Errorf("read inventory %q: %w", path, err)
		}
		for _, formula := range inventory.Formulae {
			if formula.Name == "" || formula.InstalledVersion == "" {
				continue
			}
			formulae = append(formulae, formula)
		}
	}
	return formulae, nil
}

func readJSON(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func dedupeFormulae(formulae []FormulaInventoryRow) []FormulaInventoryRow {
	seen := make(map[string]bool)
	var unique []FormulaInventoryRow
	for _, formula := range formulae {
		key := formula.Name + "\x00" + formula.InstalledVersion
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, formula)
	}
	return unique
}

func queryOSV(client *http.Client, url string, queries []osvQuery) ([]osvResult, error) {
	if len(queries) == 0 {
		return nil, nil
	}
	var results []osvResult
	for start := 0; start < len(queries); start += 1000 {
		end := min(start+1000, len(queries))
		body, err := json.Marshal(osvBatchRequest{Queries: queries[start:end]})
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return nil, fmt.Errorf("OSV returned HTTP %d", resp.StatusCode)
		}
		var batch osvBatchResponse
		if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
			return nil, err
		}
		results = append(results, batch.Results...)
	}
	return results, nil
}

func fixedVersion(vulns []osvVulnerability, mapping OSVMapping) string {
	var versions []string
	for _, vuln := range vulns {
		for _, affected := range vuln.Affected {
			if affected.Package.Ecosystem != mapping.Ecosystem || affected.Package.Name != mapping.Name {
				continue
			}
			for _, r := range affected.Ranges {
				for _, event := range r.Events {
					if event.Fixed != "" {
						versions = append(versions, event.Fixed)
					}
				}
			}
		}
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		return ""
	}
	return versions[len(versions)-1]
}

func vulnerabilityIDs(vulns []osvVulnerability) []string {
	ids := make([]string, 0, len(vulns))
	for _, vuln := range vulns {
		if vuln.ID != "" {
			ids = append(ids, vuln.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func writeMunkiPkginfo(opts MunkiPkginfoOptions, formula mappedFormula, vulns []osvVulnerability, fixedVersion string) (string, error) {
	name := "homebrew-vuln-" + formula.Formula.Name
	values := map[string]interface{}{
		"name":                           name,
		"display_name":                   "Homebrew vulnerability: " + formula.Formula.Name,
		"description":                    fmt.Sprintf("Updates vulnerable Homebrew formula %s to at least %s.", formula.Formula.Name, fixedVersion),
		"version":                        fixedVersion,
		"catalogs":                       opts.Catalogs,
		"unattended_install":             true,
		"installer_type":                 "nopkg",
		"installcheck_script":            fmt.Sprintf("#!/bin/bash\n%q check --min-version=%q --skip-not-installed --munki-installcheck %q\n", opts.CarafePath, fixedVersion, formula.Formula.Name),
		"install_script":                 fmt.Sprintf("#!/bin/bash\n%q upgrade --min-version=%q %q\n", opts.CarafePath, fixedVersion, formula.Formula.Name),
		"minimum_os_version":             "11.0",
		"developer":                      "Homebrew",
		"category":                       "Security",
		"notes":                          fmt.Sprintf("Generated from OSV data for %s:%s.", formula.Mapping.Ecosystem, formula.Mapping.Name),
		"carafe_homebrew_formula":        formula.Formula.Name,
		"carafe_installed_versions_seen": []string{formula.Formula.InstalledVersion},
		"osv_package_ecosystem":          formula.Mapping.Ecosystem,
		"osv_package_name":               formula.Mapping.Name,
		"osv_vulnerability_ids":          vulnerabilityIDs(vulns),
		"cve_ids":                        cveIDs(vulns),
	}
	path := filepath.Join(opts.OutputDir, name+".plist")
	return path, os.WriteFile(path, []byte(plist(values)), 0644)
}

func cveIDs(vulns []osvVulnerability) []string {
	seen := make(map[string]bool)
	for _, vuln := range vulns {
		for _, alias := range vuln.Aliases {
			if strings.HasPrefix(alias, "CVE-") {
				seen[alias] = true
			}
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func plist(values map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b.WriteString("\t<key>")
		xml.EscapeText(&b, []byte(key))
		b.WriteString("</key>\n")
		writePlistValue(&b, values[key])
	}
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func writePlistValue(b *strings.Builder, value interface{}) {
	switch v := value.(type) {
	case bool:
		if v {
			b.WriteString("\t<true/>\n")
		} else {
			b.WriteString("\t<false/>\n")
		}
	case []string:
		b.WriteString("\t<array>\n")
		for _, item := range v {
			b.WriteString("\t\t<string>")
			xml.EscapeText(b, []byte(item))
			b.WriteString("</string>\n")
		}
		b.WriteString("\t</array>\n")
	default:
		b.WriteString("\t<string>")
		xml.EscapeText(b, []byte(fmt.Sprint(v)))
		b.WriteString("</string>\n")
	}
}

func printSummary(result MunkiPkginfoResult) {
	fmt.Printf("Generated %d pkginfo files\n", len(result.PkginfoPaths))
	fmt.Printf("Unknown formulae: %d\n", len(result.UnknownFormulae))
	fmt.Printf("Unremediable vulnerabilities: %d\n", len(result.UnremediableVulnerabilities))
}
