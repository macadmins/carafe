package brew

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"

	_ "embed"

	"github.com/macadmins/carafe/cudo"
	"github.com/macadmins/carafe/exec"
	"github.com/macadmins/carafe/shell/testshell"
)

//go:embed info.json
var TestInfoOutput string

//go:embed info_all.json
var TestInfoAllOutput string

//go:embed info_installed.json
var TestInfoInstalledOutput string

//go:embed info_not_installed.json
var TestInfoNotInstalledOutput string

func TestAllInfo(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		runError    error
		expectError bool
	}{
		{
			name:        "installed",
			output:      TestInfoInstalledOutput,
			runError:    nil,
			expectError: false,
		},
		{
			name:        "not installed",
			output:      TestInfoNotInstalledOutput,
			runError:    nil,
			expectError: false,
		},
		{
			name:        "failure",
			output:      "No available formula with the name \"htop\". Did you mean somethingelse?",
			runError:    fmt.Errorf("run error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := exec.CarafeConfig{
				Arch: "arm64",
				CUSudo: &cudo.CUSudo{
					CurrentUser: "testuser",
					Platform:    "darwin",
					UserHome:    "/Users/testuser",
				},
			}

			if tt.runError != nil {
				c.CUSudo.Executor = testshell.NewExecutor(testshell.AlwaysError(tt.runError))
			} else {
				c.CUSudo.Executor = testshell.OutputExecutor(tt.output)
			}
			err := AllInfo(c)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestInfo(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		item        string
		runError    error
		expectError bool
	}{
		{
			name:        "success",
			output:      TestInfoOutput,
			item:        "htop",
			runError:    nil,
			expectError: false,
		},
		{
			name:        "failure",
			item:        "htop",
			output:      "No available formula with the name \"htop\". Did you mean somethingelse?",
			runError:    fmt.Errorf("run error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := exec.CarafeConfig{
				Arch: "arm64",
				CUSudo: &cudo.CUSudo{
					CurrentUser: "testuser",
					Platform:    "darwin",
					UserHome:    "/Users/testuser",
				},
			}

			if tt.runError != nil {
				c.CUSudo.Executor = testshell.NewExecutor(testshell.AlwaysError(tt.runError))
			} else {
				c.CUSudo.Executor = testshell.OutputExecutor(tt.output)
			}
			err := Info(c, tt.item)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestInfoOutputFunc(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		item        string
		runError    error
		expectError bool
	}{
		{
			name:        "success",
			output:      TestInfoOutput,
			item:        "htop",
			runError:    nil,
			expectError: false,
		},
		{
			name:        "failure",
			item:        "htop",
			output:      "No available formula with the name \"htop\". Did you mean somethingelse?",
			runError:    fmt.Errorf("run error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := exec.CarafeConfig{
				Arch: "arm64",
				CUSudo: &cudo.CUSudo{
					CurrentUser: "testuser",
					Platform:    "darwin",
					UserHome:    "/Users/testuser",
				},
			}

			if tt.runError != nil {
				c.CUSudo.Executor = testshell.NewExecutor(testshell.AlwaysError(tt.runError))
			} else {
				c.CUSudo.Executor = testshell.OutputExecutor(tt.output)
			}
			out, err := infoOutput(c, tt.item)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.output, out)
		})
	}
}

func TestInstalled(t *testing.T) {
	tests := []struct {
		name        string
		output      bool
		stdout      string
		runError    error
		expectError bool
	}{
		{
			name:        "installed",
			stdout:      TestInfoInstalledOutput,
			output:      true,
			runError:    nil,
			expectError: false,
		},
		{
			name:        "not installed",
			stdout:      TestInfoNotInstalledOutput,
			output:      false,
			runError:    nil,
			expectError: false,
		},
		{
			name:        "failure",
			output:      true,
			stdout:      "No available formula with the name \"htop\". Did you mean somethingelse?",
			runError:    fmt.Errorf("run error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.stdout == "" {
				t.Fatal("empty stdout")
			}

			installed, err := installed(tt.stdout)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.output {
				assert.True(t, installed)
			} else {
				assert.False(t, installed)
			}

		})
	}
}

func TestIsInstalled(t *testing.T) {
	tests := []struct {
		name        string
		item        string
		stdout      string
		output      bool
		runError    error
		expectError bool
	}{
		{
			name:        "installed",
			item:        "htop",
			stdout:      TestInfoInstalledOutput,
			output:      true,
			runError:    nil,
			expectError: false,
		},
		{
			name:        "not installed",
			item:        "htop",
			stdout:      TestInfoNotInstalledOutput,
			output:      false,
			runError:    nil,
			expectError: false,
		},
		{
			name:        "failure",
			item:        "htop",
			stdout:      "No available formula with the name \"htop\". Did you mean somethingelse?",
			output:      false,
			runError:    fmt.Errorf("run error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := exec.CarafeConfig{
				Arch: "arm64",
				CUSudo: &cudo.CUSudo{
					CurrentUser: "testuser",
					Platform:    "darwin",
					UserHome:    "/Users/testuser",
				},
			}

			if tt.runError != nil {
				c.CUSudo.Executor = testshell.NewExecutor(testshell.AlwaysError(tt.runError))
			} else {
				c.CUSudo.Executor = testshell.OutputExecutor(tt.stdout)
			}

			installed, err := IsInstalled(c, tt.item)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.output, installed)
		})
	}
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		expected    string
		expectError bool
	}{
		{
			name:        "valid JSON with version",
			output:      `[{ "installed": [{ "version": "1.2.3" }] }]`,
			expected:    "1.2.3",
			expectError: false,
		},
		{
			name:        "valid JSON without installed version",
			output:      `[{ "installed": [] }]`,
			expected:    "",
			expectError: false,
		},
		{
			name:        "invalid JSON",
			output:      `{ "installed": [}`,
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty JSON array",
			output:      `[]`,
			expected:    "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := getVersion(tt.output)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestInstalledVersion(t *testing.T) {
	tests := []struct {
		name        string
		item        string
		stdout      string
		expected    string
		runError    error
		expectError bool
	}{
		{
			name:        "installed version",
			item:        "htop",
			stdout:      `[{ "installed": [{ "version": "1.2.3" }] }]`,
			expected:    "1.2.3",
			runError:    nil,
			expectError: false,
		},
		{
			name:        "not installed",
			item:        "htop",
			stdout:      `[{ "installed": [] }]`,
			expected:    "",
			runError:    nil,
			expectError: false,
		},
		{
			name:        "command error",
			item:        "htop",
			stdout:      "",
			expected:    "",
			runError:    fmt.Errorf("run error"),
			expectError: true,
		},
		{
			name:        "invalid JSON",
			item:        "htop",
			stdout:      `{ "installed": [}`,
			expected:    "",
			runError:    nil,
			expectError: true,
		},
		{
			name:        "empty JSON array",
			item:        "htop",
			stdout:      `[]`,
			expected:    "",
			runError:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := exec.CarafeConfig{
				Arch: "arm64",
				CUSudo: &cudo.CUSudo{
					CurrentUser: "testuser",
					Platform:    "darwin",
					UserHome:    "/Users/testuser",
				},
			}

			if tt.runError != nil {
				c.CUSudo.Executor = testshell.NewExecutor(testshell.AlwaysError(tt.runError))
			} else {
				c.CUSudo.Executor = testshell.OutputExecutor(tt.stdout)
			}

			version, err := InstalledVersion(c, tt.item)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestStripBrewRevision(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no revision",
			input:    "2.52.0",
			expected: "2.52.0",
		},
		{
			name:     "single digit revision",
			input:    "2.52.0_1",
			expected: "2.52.0",
		},
		{
			name:     "multi digit revision",
			input:    "2.52.0_12",
			expected: "2.52.0",
		},
		{
			name:     "long version with revision",
			input:    "2026.01.12.00_1",
			expected: "2026.01.12.00",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, stripBrewRevision(tt.input))
		})
	}
}

func TestVersionMeetsOrExceedsMinimum(t *testing.T) {
	tests := []struct {
		name           string
		item           string
		minimumVersion string
		stdout         string
		expected       bool
		runError       error
		expectError    bool
	}{
		{
			name:           "installed version meets minimum",
			item:           "htop",
			minimumVersion: "1.2.0",
			stdout:         `[{ "installed": [{ "version": "1.2.3" }] }]`,
			expected:       true,
			runError:       nil,
			expectError:    false,
		},
		{
			name:           "installed version does not meet minimum",
			item:           "htop",
			minimumVersion: "1.2.4",
			stdout:         `[{ "installed": [{ "version": "1.2.3" }] }]`,
			expected:       false,
			runError:       nil,
			expectError:    false,
		},
		{
			name:           "installed version with brew revision meets minimum",
			item:           "git",
			minimumVersion: "2.45.2",
			stdout:         `[{ "installed": [{ "version": "2.52.0_1" }] }]`,
			expected:       true,
			runError:       nil,
			expectError:    false,
		},
		{
			name:           "installed version with brew revision does not meet minimum",
			item:           "git",
			minimumVersion: "2.53.0",
			stdout:         `[{ "installed": [{ "version": "2.52.0_1" }] }]`,
			expected:       false,
			runError:       nil,
			expectError:    false,
		},
		{
			name:           "not installed",
			item:           "htop",
			minimumVersion: "1.2.0",
			stdout:         `[{ "installed": [] }]`,
			expected:       true,
			runError:       nil,
			expectError:    false,
		},
		{
			name:           "command error",
			item:           "htop",
			minimumVersion: "1.2.0",
			stdout:         "",
			expected:       true,
			runError:       fmt.Errorf("run error"),
			expectError:    true,
		},
		{
			name:           "invalid JSON",
			item:           "htop",
			minimumVersion: "1.2.0",
			stdout:         `{ "installed": [}`,
			expected:       true,
			runError:       nil,
			expectError:    true,
		},
		{
			name:           "empty JSON array",
			item:           "htop",
			minimumVersion: "1.2.0",
			stdout:         `[]`,
			expected:       true,
			runError:       nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := exec.CarafeConfig{
				Arch: "arm64",
				CUSudo: &cudo.CUSudo{
					CurrentUser: "testuser",
					Platform:    "darwin",
					UserHome:    "/Users/testuser",
				},
			}

			if tt.runError != nil {
				c.CUSudo.Executor = testshell.NewExecutor(testshell.AlwaysError(tt.runError))
			} else {
				c.CUSudo.Executor = testshell.OutputExecutor(tt.stdout)
			}

			result, err := VersionMeetsOrExceedsMinimum(c, tt.item, tt.minimumVersion)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareBrewVersions(t *testing.T) {
	tests := []struct {
		installed string
		minimum   string
		expected  bool
	}{
		// Plain versions and revisions keep working.
		{"2.52.0", "2.52.0", true},
		{"2.52.0_1", "2.52.0", true},
		{"2.51.9", "2.52.0", false},
		// A letter suffix is a later release, not a pre-release (openssh, openssl@1.1, tzdata style).
		{"10.2p1", "10.2", true},
		{"10.2p1", "10.2p1", true},
		{"10.2p2", "10.2p1", true},
		{"10.2p1", "10.2p2", false},
		{"10.2p10", "10.2p9", true},
		{"10.1p1", "10.2", false},
		{"9.9p2", "10.0p1", false},
		{"10.0p1", "9.9p2", true},
		{"1.1.1w", "1.1.1", true},
		{"1.1.1w", "1.1.1x", false},
		{"2025b", "2025a", true},
		{"2025a", "2025b", false},
		{"10.2p1_1", "10.2p1", true},
		{"1.0.2zk", "1.0.2u", true},
		{"1.0b", "1.0", true},
		// Other letters with a number are later releases, as in Homebrew.
		{"4.5ga6", "4.5", true},
		{"2.1c1", "2.1", true},
		{"1.0.0dev1", "1.0.0", true},
		{"1.0.0post1", "1.0.0", true},
		{"3.1jupp41", "3.1", true},
		// Suffix numbers of any length compare as numbers.
		{"3.3.0p99999999999999999999", "3.3.0p99999999999999999998", true},
		{"3.3.0p9", "3.3.0p10", false},
		{"3.3.0p010", "3.3.0p9", true},
		// Pre-release numbers compare as numbers, and pre-release kinds in
		// Homebrew's order: alpha (a) < beta (b) < preview < pre < rc.
		{"3.0.0rc10", "3.0.0rc9", true},
		{"1.0beta10", "1.0beta9", true},
		{"1.0b10", "1.0b9", true},
		{"3.14.0a10", "3.14.0a9", true},
		{"1.0a10", "1.0b1", false},
		{"1.0a1", "1.0alpha1", true},
		{"1.0alpha1", "1.0a1", true},
		{"1.0pre1", "1.0preview1", true},
		{"1.0preview1", "1.0pre1", false},
		{"1.0rc1", "1.0pre9", true},
		{"1.0rc", "1.0rc0", true},
		{"1.0beta", "1.0beta1", false},
		// A leading "v" or a "-" before the suffix does not change the order.
		{"10.2p1", "v10.2p2", false},
		{"10.2p2", "v10.2p2", true},
		{"3.0.0rc9", "v3.0.0rc10", false},
		{"1.0.0-rc3", "1.0.0rc2", true},
		{"1.0.0rc1", "1.0.0-rc2", false},
		{"1.0-p1", "1.0", true},
		// Named pre-releases stay pre-releases.
		{"1.0.0preview1", "1.0.0", false},
		{"1.0.0pre", "1.0.0", false},
		{"1.0.0b1", "1.0.0", false},
		{"3.14.0a2", "3.14.0", false},
		{"1.0.0b2", "1.0.0b1", true},
		{"1.0.0", "1.0.0b1", true},
		{"3.0.0rc1", "3.0.0", false},
		{"3.0.0-rc1", "3.0.0", false},
		{"3.0.0beta2", "3.0.0beta1", true},
	}

	for _, tt := range tests {
		t.Run(tt.installed+">="+tt.minimum, func(t *testing.T) {
			got, err := versionAtLeast(tt.installed, tt.minimum)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCompareBrewVersionsInvalid(t *testing.T) {
	_, err := versionAtLeast("HEAD-1234567", "1.0")
	assert.Error(t, err)
	// A revision in the minimum is not silently dropped.
	_, err = versionAtLeast("2.50.1", "2.50.1_1")
	assert.Error(t, err)
	_, err = versionAtLeast("1.0", "not a version")
	assert.Error(t, err)
}

// TestParseBrewVersionAcceptsSameAsGoVersion pins that parseBrewVersion accepts
// exactly the strings that go-version accepts. A caller that validates a
// minimum version with go-version can then rely on the comparison never
// failing for a value that passed validation.
func TestParseBrewVersionAcceptsSameAsGoVersion(t *testing.T) {
	inputs := []string{
		"1.0", "10.2p1", "1.1.1w", "2025b", "4.5ga6", "3.0.0rc10", "1.0b1", "1.0beta",
		"1.0-rc1", "1.0-p1", "1.0P1", "v1.2.3", "v10.2p2", "v1.0-rc1", "1.0-a", "1.0--rc1", "v", "vv1.0", "1.2.3+meta", "3.3.0p99999999999999999999",
		"", "v2.x", "HEAD-1234567", "2.50.1_1", "0.99.beta20", "r3222", "1..0", "1.0 p1", "1.0p1x",
	}
	for _, in := range inputs {
		_, brewErr := parseBrewVersion(in)
		_, goErr := version.NewVersion(in)
		assert.Equal(t, goErr == nil, brewErr == nil, "input %q: go-version err=%v, parseBrewVersion err=%v", in, goErr, brewErr)
	}
}
