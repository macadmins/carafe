package brew

import (
	"testing"

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
