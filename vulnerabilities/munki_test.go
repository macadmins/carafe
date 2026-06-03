package vulnerabilities

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMunkiPkginfos(t *testing.T) {
	tmp := t.TempDir()
	inventoryPath := filepath.Join(tmp, "inventory.json")
	mappingPath := filepath.Join(tmp, "mapping.json")
	outputDir := filepath.Join(tmp, "pkginfos")
	require.NoError(t, os.WriteFile(inventoryPath, []byte(`{
  "formulae": [
    {"name": "git", "full_name": "git", "installed_version": "2.50.0"},
    {"name": "git", "full_name": "git", "installed_version": "2.50.0"},
    {"name": "unknown", "full_name": "unknown", "installed_version": "1.0.0"}
  ]
}`), 0600))
	require.NoError(t, os.WriteFile(mappingPath, []byte(`{
  "git": {"ecosystem": "Packagist", "name": "git/git"}
}`), 0600))

	var requestBody osvBatchRequest
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&requestBody))
		return jsonResponse(`{
  "results": [
    {
      "vulns": [
        {
          "id": "OSV-2026-1",
          "aliases": ["CVE-2026-0001"],
          "affected": [
            {
              "package": {"ecosystem": "Packagist", "name": "git/git"},
              "ranges": [{"events": [{"introduced": "0"}, {"fixed": "2.50.1"}]}]
            }
          ]
        }
      ]
    }
  ]
}`)
	})}

	result, err := GenerateMunkiPkginfos(MunkiPkginfoOptions{
		InventoryPaths: []string{inventoryPath},
		MappingPath:    mappingPath,
		OutputDir:      outputDir,
		Catalogs:       []string{"testing", "production"},
		CarafePath:     "/usr/local/bin/carafe",
		OSVURL:         "https://osv.example.test/v1/querybatch",
		HTTPClient:     client,
	})

	require.NoError(t, err)
	require.Len(t, requestBody.Queries, 1)
	assert.Equal(t, "2.50.0", requestBody.Queries[0].Version)
	assert.Equal(t, "Packagist", requestBody.Queries[0].Package.Ecosystem)
	require.Len(t, result.UnknownFormulae, 1)
	assert.Equal(t, "unknown", result.UnknownFormulae[0].Name)
	require.Len(t, result.PkginfoPaths, 1)
	data, err := os.ReadFile(result.PkginfoPaths[0])
	require.NoError(t, err)
	pkginfo := string(data)
	assert.Contains(t, pkginfo, "<string>homebrew-vuln-git</string>")
	assert.Contains(t, pkginfo, "<string>2.50.1</string>")
	assert.Contains(t, pkginfo, "/usr/local/bin/carafe&#34; check --min-version=&#34;2.50.1&#34;")
	assert.Contains(t, pkginfo, "<string>CVE-2026-0001</string>")
}

func TestGenerateMunkiPkginfosSkipsUnremediable(t *testing.T) {
	tmp := t.TempDir()
	inventoryPath := filepath.Join(tmp, "inventory.json")
	mappingPath := filepath.Join(tmp, "mapping.json")
	require.NoError(t, os.WriteFile(inventoryPath, []byte(`{
  "formulae": [{"name": "git", "installed_version": "2.50.0"}]
}`), 0600))
	require.NoError(t, os.WriteFile(mappingPath, []byte(`{
  "git": {"ecosystem": "Packagist", "name": "git/git"}
}`), 0600))

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{
  "results": [{"vulns": [{"id": "OSV-2026-1", "affected": []}]}]
}`)
	})}

	result, err := GenerateMunkiPkginfos(MunkiPkginfoOptions{
		InventoryPaths: []string{inventoryPath},
		MappingPath:    mappingPath,
		OutputDir:      filepath.Join(tmp, "pkginfos"),
		OSVURL:         "https://osv.example.test/v1/querybatch",
		HTTPClient:     client,
	})

	require.NoError(t, err)
	assert.Empty(t, result.PkginfoPaths)
	require.Len(t, result.UnremediableVulnerabilities, 1)
	assert.Equal(t, "OSV-2026-1", strings.Join(result.UnremediableVulnerabilities[0].VulnerabilityIDs, ""))
}

func TestGenerateMunkiPkginfosRequiresInputs(t *testing.T) {
	_, err := GenerateMunkiPkginfos(MunkiPkginfoOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--inventory")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(body string) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}, nil
}
