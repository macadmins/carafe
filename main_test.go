package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/macadmins/carafe/cudo"
	"github.com/macadmins/carafe/exec"
	"github.com/macadmins/carafe/shell/testshell"
)

func TestValidateFormulaArg_AllowsValid(t *testing.T) {
	valid := []string{
		"openssl",
		"openssl@1.1",
		"libxml++",
		"qt-5",
		"foo_bar.baz",
		"gcc@12",
		"a",
		strings.Repeat("a", 128),
	}
	for _, v := range valid {
		assert.NoError(t, validateFormulaArg(v))
		assert.True(t, formulaRe.MatchString(v))
	}
}

func TestValidateFormulaArg_RejectsInvalid(t *testing.T) {
	invalid := []string{
		"",
		"homebrew/cask/vlc",
		"foo bar",
		"naïve",
		"foo:bar",
		"abc!",
		"a\nb",
		strings.Repeat("a", 129),
	}
	for _, v := range invalid {
		assert.Error(t, validateFormulaArg(v))
		assert.False(t, formulaRe.MatchString(v))
	}
}

func TestFormulaRe_DoesNotPartiallyMatch(t *testing.T) {
	assert.False(t, formulaRe.MatchString("valid@1.0!")) // '!' not allowed anywhere
	assert.False(t, formulaRe.MatchString(" valid"))     // leading space
	assert.False(t, formulaRe.MatchString("valid "))     // trailing space
}

func newMockConfig() exec.CarafeConfig {
	return exec.CarafeConfig{
		Arch: "arm64",
		CUSudo: &cudo.CUSudo{
			CurrentUser: "testuser",
			Platform:    "darwin",
			OSFunc:      &cudo.MockOSFunc{},
			UserHome:    "/Users/testuser",
			Executor:    testshell.NewExecutor(),
		},
	}
}

func TestCheckCmd_InvalidCacheTTL(t *testing.T) {
	cmd := buildRootCmd(newMockConfig(), "test")
	cmd.SetArgs([]string{"check", "--cache-ttl=banana", "git"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid argument")
}

func TestCheckCmd_NoCacheFlag(t *testing.T) {
	// --no-cache should be accepted without error (brew call itself will fail
	// with the mock executor, but the flag parsing must succeed).
	cmd := buildRootCmd(newMockConfig(), "test")
	cmd.SetArgs([]string{"check", "--no-cache", "git"})
	// The mock executor has no output configured so brew.Check will error,
	// but the error must NOT be a flag-parsing error.
	err := cmd.Execute()
	if err != nil {
		assert.NotContains(t, err.Error(), "invalid argument")
		assert.NotContains(t, err.Error(), "unknown flag")
	}
}

func TestCheckCmd_CacheTTLFlag(t *testing.T) {
	// Valid duration strings must be accepted.
	for _, ttl := range []string{"30s", "2m", "1h", "0s"} {
		cmd := buildRootCmd(newMockConfig(), "test")
		cmd.SetArgs([]string{"check", "--cache-ttl=" + ttl, "--no-cache", "git"})
		err := cmd.Execute()
		if err != nil {
			assert.NotContains(t, err.Error(), "invalid argument", "TTL %q should be valid", ttl)
		}
	}
}

func TestInventoryHomebrewCmd_WritesOutputFile(t *testing.T) {
	cmd := buildRootCmd(newMockConfigWithOutput(`{
  "formulae": [
    {
      "name": "git",
      "full_name": "git",
      "tap": "homebrew/core",
      "installed": [{"version": "2.50.0", "installed_on_request": true}]
    }
  ],
  "casks": [{"token": "firefox", "version": "126.0"}]
}`), "test")
	outputPath := filepath.Join(t.TempDir(), "inventory.json")
	cmd.SetArgs([]string{"inventory", "homebrew", "--output", outputPath})

	err := cmd.Execute()

	require.NoError(t, err)
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	var inventory struct {
		Formulae []struct {
			Name             string `json:"name"`
			InstalledVersion string `json:"installed_version"`
		} `json:"formulae"`
	}
	require.NoError(t, json.Unmarshal(data, &inventory))
	require.Len(t, inventory.Formulae, 1)
	assert.Equal(t, "git", inventory.Formulae[0].Name)
	assert.Equal(t, "2.50.0", inventory.Formulae[0].InstalledVersion)
}

func TestVulnerabilitiesCmd_DoesNotInitializeBrewConfigForValidation(t *testing.T) {
	cmd := buildRootCmdWithConfig(func() (exec.CarafeConfig, error) {
		return exec.CarafeConfig{}, assert.AnError
	}, "test")
	cmd.SetArgs([]string{"vulnerabilities", "munki-pkginfos"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--inventory")
	assert.NotContains(t, err.Error(), assert.AnError.Error())
}

func newMockConfigWithOutput(outputs ...string) exec.CarafeConfig {
	c := newMockConfig()
	c.CUSudo.Executor = testshell.OutputExecutor(outputs...)
	return c
}
