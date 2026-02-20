package brew

import (
	"fmt"
	"testing"

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
