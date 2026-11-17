package brew

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/macadmins/carafe/cudo"
	"github.com/macadmins/carafe/exec"
	"github.com/macadmins/carafe/shell/testshell"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name               string
		item               string
		minVersion         string
		munkiInstallsCheck bool
		skipNotInstalled   bool
		outputs            []string
		expectedResult     int
		expectedError      bool
	}{
		{
			name:               "installed",
			item:               "htop",
			minVersion:         "",
			munkiInstallsCheck: false,
			outputs:            []string{TestInfoInstalledOutput},
			expectedResult:     0,
		},
		{
			name:               "not installed",
			item:               "htop",
			minVersion:         "",
			munkiInstallsCheck: false,
			outputs:            []string{TestInfoNotInstalledOutput},
			expectedResult:     1,
		},
		{
			name:               "meets minimum version",
			item:               "htop",
			minVersion:         "2.0.0",
			munkiInstallsCheck: false,
			outputs:            []string{TestInfoInstalledOutput, TestInfoInstalledOutput}, // Register multiple outputs
			expectedResult:     0,
		},
		{
			name:               "does not meet minimum version",
			item:               "htop",
			minVersion:         "4.0.0",
			munkiInstallsCheck: false,
			outputs:            []string{TestInfoInstalledOutput, TestInfoInstalledOutput}, // Register multiple outputs
			expectedResult:     1,
		},
		{
			name:               "munki installs check",
			item:               "htop",
			minVersion:         "",
			munkiInstallsCheck: true,
			outputs:            []string{TestInfoInstalledOutput},
			expectedResult:     1,
		},
		{
			name:               "munki installs check not installed",
			item:               "htop",
			outputs:            []string{TestInfoNotInstalledOutput},
			expectedResult:     0,
			munkiInstallsCheck: true,
		},
		{
			name:               "munki installs check meets minimum version",
			item:               "htop",
			minVersion:         "2.0.0",
			outputs:            []string{TestInfoInstalledOutput, TestInfoInstalledOutput},
			expectedResult:     1,
			munkiInstallsCheck: true,
		},
		{
			name:               "munki installs check does not meet minimum version",
			item:               "htop",
			minVersion:         "4.0.0",
			outputs:            []string{TestInfoInstalledOutput, TestInfoInstalledOutput},
			expectedResult:     0,
			munkiInstallsCheck: true,
		},
		{
			name:               "skip not installed not installed",
			item:               "htop",
			outputs:            []string{TestInfoNotInstalledOutput},
			expectedResult:     0,
			munkiInstallsCheck: false,
			skipNotInstalled:   true,
			minVersion:         "2.0.0",
		},
		{
			name:               "skip not installed installed",
			item:               "htop",
			outputs:            []string{TestInfoInstalledOutput, TestInfoInstalledOutput},
			minVersion:         "2.0.0",
			expectedResult:     0,
			munkiInstallsCheck: false,
			skipNotInstalled:   true,
		},
		{
			name:               "skip not installed set, no min version",
			item:               "htop",
			outputs:            []string{TestInfoInstalledOutput, TestInfoInstalledOutput},
			expectedResult:     1,
			munkiInstallsCheck: false,
			skipNotInstalled:   true,
			expectedError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := exec.CarafeConfig{
				Arch: "arm64",
				CUSudo: &cudo.CUSudo{
					CurrentUser: "testuser",
					Platform:    "darwin",
					OSFunc:      &cudo.MockOSFunc{},
					UserHome:    "/Users/testuser",
					Executor:    testshell.OutputExecutor(tt.outputs...),
				},
			}

			// Call the function under test (cacheTTL=0 disables caching)
			result, err := Check(c, tt.item, tt.minVersion, tt.munkiInstallsCheck, tt.skipNotInstalled, 0)

			// Assert the error
			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Assert the result
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestCheckErrorsFailClosed makes sure an error never yields the "compliant"
// exit code. In --munki-installcheck mode exit 1 means "nothing to do", so a
// broken brew or an unreadable version must exit 0 instead.
func TestCheckErrorsFailClosed(t *testing.T) {
	unparseable := `[{"name":"git","installed":[{"version":"HEAD-1234567"}]}]`

	tests := []struct {
		name               string
		munkiInstallsCheck bool
		opts               []testshell.ExecutorOption
		expectedResult     int
	}{
		{
			name:               "brew fails, munki mode",
			munkiInstallsCheck: true,
			opts:               []testshell.ExecutorOption{testshell.AlwaysError(errors.New("brew failed"))},
			expectedResult:     0,
		},
		{
			name:               "brew fails, plain mode",
			munkiInstallsCheck: false,
			opts:               []testshell.ExecutorOption{testshell.AlwaysError(errors.New("brew failed"))},
			expectedResult:     1,
		},
		{
			name:               "unparseable installed version, munki mode",
			munkiInstallsCheck: true,
			opts:               []testshell.ExecutorOption{testshell.WithStdout(unparseable)},
			expectedResult:     0,
		},
		{
			name:               "unparseable installed version, plain mode",
			munkiInstallsCheck: false,
			opts:               []testshell.ExecutorOption{testshell.WithStdout(unparseable)},
			expectedResult:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := exec.CarafeConfig{
				Arch: "arm64",
				CUSudo: &cudo.CUSudo{
					CurrentUser: "testuser",
					Platform:    "darwin",
					OSFunc:      &cudo.MockOSFunc{},
					UserHome:    "/Users/testuser",
					Executor:    testshell.NewExecutor(tt.opts...),
				},
			}

			result, err := Check(c, "git", "2.50.1", tt.munkiInstallsCheck, true, 0)

			assert.Error(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestCheckUnknownFormulaAndUsageErrors(t *testing.T) {
	unknown := testshell.AlwaysError(errors.New(`Error: No available formula with the name "nosuch".`))

	tests := []struct {
		name               string
		item               string
		minVersion         string
		munkiInstallsCheck bool
		skipNotInstalled   bool
		cacheTTL           time.Duration
		rackPresent        bool // a keg for the item exists in the Cellar
		opts               []testshell.ExecutorOption
		expectedResult     int
		expectedError      bool
		expectedUsage      bool
	}{
		{
			// brew prints the name in lower case.
			name:               "unknown formula in mixed case, skip not installed, munki mode",
			item:               "NoSuch",
			minVersion:         "1.0",
			munkiInstallsCheck: true,
			skipNotInstalled:   true,
			opts:               []testshell.ExecutorOption{unknown},
			expectedResult:     1,
		},
		{
			name:               "unknown formula through the cache, skip not installed, munki mode",
			minVersion:         "1.0",
			munkiInstallsCheck: true,
			skipNotInstalled:   true,
			cacheTTL:           time.Minute,
			opts:               []testshell.ExecutorOption{unknown},
			expectedResult:     1,
		},
		{
			// A formula from a tap that is not tapped cannot be installed: not installed.
			name:               "formula from an untapped tap, skip not installed, munki mode",
			item:               "user/tap/nosuch",
			minVersion:         "1.0",
			munkiInstallsCheck: true,
			skipNotInstalled:   true,
			opts: []testshell.ExecutorOption{testshell.AlwaysError(errors.New(
				"Error: No available formula with the name \"user/tap/nosuch\".\nThis command requires the tap user/tap."))},
			expectedResult: 1,
		},
		{
			// The formula file exists but is broken: fail closed.
			name:               "broken formula file, skip not installed, munki mode",
			minVersion:         "1.0",
			munkiInstallsCheck: true,
			skipNotInstalled:   true,
			opts: []testshell.ExecutorOption{testshell.AlwaysError(errors.New(
				"Error: No available formula with the name \"nosuch\".\nIn formula file: /opt/homebrew/Library/Taps/user/homebrew-tap/nosuch.rb\nExpected to find class Nosuch, but found no classes."))},
			expectedResult: 0,
			expectedError:  true,
		},
		{
			// A formula removed from homebrew-core must not become "install needed" fleet-wide.
			name:               "unknown formula, skip not installed, munki mode",
			minVersion:         "1.0",
			munkiInstallsCheck: true,
			skipNotInstalled:   true,
			opts:               []testshell.ExecutorOption{unknown},
			expectedResult:     1,
		},
		{
			name:             "unknown formula, skip not installed, plain mode",
			minVersion:       "1.0",
			skipNotInstalled: true,
			opts:             []testshell.ExecutorOption{unknown},
			expectedResult:   0,
		},
		{
			name:               "unknown formula without skip not installed, munki mode",
			minVersion:         "1.0",
			munkiInstallsCheck: true,
			opts:               []testshell.ExecutorOption{unknown},
			expectedResult:     0,
			expectedError:      true,
		},
		{
			// brew names a different formula: the requested one is not unknown, so fail closed.
			name:               "other formula unknown, skip not installed, munki mode",
			minVersion:         "1.0",
			munkiInstallsCheck: true,
			skipNotInstalled:   true,
			opts: []testshell.ExecutorOption{testshell.AlwaysError(
				errors.New(`Error: No available formula with the name "bar" (dependency of nosuch).`))},
			expectedResult: 0,
			expectedError:  true,
		},
		{
			// The formula was removed from its tap (or the tap was untapped), but
			// the keg is still in the Cellar: installed, so fail closed.
			name:               "unknown formula but keg in the Cellar, skip not installed, munki mode",
			minVersion:         "1.0",
			munkiInstallsCheck: true,
			skipNotInstalled:   true,
			rackPresent:        true,
			opts:               []testshell.ExecutorOption{unknown},
			expectedResult:     0,
			expectedError:      true,
		},
		{
			name:             "invalid min version, plain mode",
			minVersion:       "v2.x",
			skipNotInstalled: true,
			opts:             []testshell.ExecutorOption{testshell.WithStdout(TestInfoInstalledOutput)},
			expectedResult:   1,
			expectedError:    true,
			expectedUsage:    true,
		},
		{
			name:               "min version with a revision is a usage error",
			minVersion:         "3.3.0_1",
			munkiInstallsCheck: true,
			skipNotInstalled:   true,
			opts:               []testshell.ExecutorOption{testshell.WithStdout(TestInfoInstalledOutput)},
			expectedResult:     0,
			expectedError:      true,
			expectedUsage:      true,
		},
		{
			name:               "invalid min version, munki mode",
			minVersion:         "v2.x",
			munkiInstallsCheck: true,
			skipNotInstalled:   true,
			opts:               []testshell.ExecutorOption{testshell.WithStdout(TestInfoInstalledOutput)},
			expectedResult:     0,
			expectedError:      true,
			expectedUsage:      true,
		},
		{
			name:               "skip not installed without min version, munki mode",
			munkiInstallsCheck: true,
			skipNotInstalled:   true,
			opts:               []testshell.ExecutorOption{testshell.WithStdout(TestInfoInstalledOutput)},
			expectedResult:     0,
			expectedError:      true,
			expectedUsage:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := exec.CarafeConfig{
				Arch: "arm64",
				CUSudo: &cudo.CUSudo{
					CurrentUser: "testuser",
					Platform:    "darwin",
					OSFunc: &cudo.MockOSFunc{StatFunc: func(name string) (os.FileInfo, error) {
						if strings.Contains(name, "/Cellar/") && !tt.rackPresent {
							return nil, os.ErrNotExist
						}
						return nil, nil
					}},
					UserHome: "/Users/testuser",
					Executor: testshell.NewExecutor(tt.opts...),
				},
			}

			item := tt.item
			if item == "" {
				item = "nosuch"
			}
			result, err := Check(c, item, tt.minVersion, tt.munkiInstallsCheck, tt.skipNotInstalled, tt.cacheTTL)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedUsage, errors.Is(err, ErrUsage))
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
