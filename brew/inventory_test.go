package brew

import (
	"encoding/json"
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

const testBrewInfoV2Installed = `{
  "formulae": [
    {
      "name": "git",
      "full_name": "git",
      "tap": "homebrew/core",
      "homepage": "https://git-scm.com",
      "license": "GPL-2.0-only",
      "urls": {
        "stable": {
          "url": "https://www.kernel.org/pub/software/scm/git/git-2.50.1.tar.xz",
          "checksum": "abc123"
        }
      },
      "installed": [
        {
          "version": "2.50.0",
          "installed_on_request": true
        }
      ],
      "outdated": true
    },
    {
      "name": "not-installed",
      "full_name": "not-installed",
      "tap": "homebrew/core",
      "installed": []
    }
  ],
  "casks": [
    {
      "token": "firefox",
      "version": "126.0"
    }
  ]
}`

func TestFormulaInventoryFromOutput(t *testing.T) {
	generatedAt := time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC)

	inventory, err := formulaInventoryFromOutput(testBrewInfoV2Installed, "/opt/homebrew/bin/brew", "arm64", generatedAt)

	require.NoError(t, err)
	assert.Equal(t, "1", inventory.SchemaVersion)
	assert.Equal(t, "2026-06-01T12:30:00Z", inventory.GeneratedAt)
	assert.Equal(t, "/opt/homebrew/bin/brew", inventory.BrewPath)
	assert.Equal(t, "arm64", inventory.Arch)
	require.Len(t, inventory.Formulae, 1)
	assert.Equal(t, FormulaInventoryRow{
		Name:               "git",
		FullName:           "git",
		Tap:                "homebrew/core",
		InstalledVersion:   "2.50.0",
		InstalledOnRequest: true,
		SourceURL:          "https://www.kernel.org/pub/software/scm/git/git-2.50.1.tar.xz",
		Homepage:           "https://git-scm.com",
		License:            "GPL-2.0-only",
		Checksum:           "abc123",
		Outdated:           true,
	}, inventory.Formulae[0])
}

func TestFormulaInventoryOutput(t *testing.T) {
	c := exec.CarafeConfig{
		Arch: "arm64",
		CUSudo: &cudo.CUSudo{
			CurrentUser: "testuser",
			Platform:    "darwin",
			OSFunc:      &cudo.MockOSFunc{},
			UserHome:    "/Users/testuser",
			Executor:    testshell.OutputExecutor(testBrewInfoV2Installed),
		},
	}

	inventory, err := FormulaInventoryOutput(c, time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC))

	require.NoError(t, err)
	require.Len(t, inventory.Formulae, 1)
	assert.Equal(t, "git", inventory.Formulae[0].Name)
}

func TestWriteFormulaInventoryData(t *testing.T) {
	inventory := FormulaInventory{
		SchemaVersion: "1",
		GeneratedAt:   "2026-06-01T12:30:00Z",
		BrewPath:      "/opt/homebrew/bin/brew",
		Arch:          "arm64",
		Formulae: []FormulaInventoryRow{
			{Name: "git", InstalledVersion: "2.50.0"},
		},
	}
	path := filepath.Join(t.TempDir(), "nested", "homebrew_formulae.json")

	err := WriteFormulaInventoryData(path, inventory)

	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var got FormulaInventory
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, inventory, got)
}

func TestFormulaInventoryFromOutputRejectsInvalidJSON(t *testing.T) {
	_, err := formulaInventoryFromOutput("{", "/opt/homebrew/bin/brew", "arm64", time.Now())

	assert.Error(t, err)
}
