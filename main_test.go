package main

import (
	"bytes"
	"errors"
	"fmt"
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
		"homebrew/core/openssl@3",
		"someuser/sometap/foo",
		"some-user/some_tap/libxml++",
		"user/tap.name/foo",
		"homebrew/caskify/foo", // not an official cask tap
		"homebrew/caskroom/foo",
		"someuser/cask/foo",
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
	}
}

func TestValidateFormulaArg_RejectsInvalidTapQualified(t *testing.T) {
	invalid := []string{
		"homebrew/cask/vlc",
		"Homebrew/cask/vlc",
		"homebrew/cask-versions/firefox@beta",
		"homebrew/cask-fonts/font-hack",
		"homebrew/foo",     // tap without formula
		"a/b/c/d",          // too many segments
		"/core/foo",        // empty user
		"homebrew//foo",    // empty repo
		"homebrew/core/",   // empty formula
		"-user/tap/foo",    // user must start with alphanumeric
		"user/.tap/foo",    // repo must start with alphanumeric
		"user/../foo",      // path traversal
		"user/tap/foo bar", // whitespace
		"user/tap/foo:bar", // invalid character
		strings.Repeat("a", 65) + "/tap/foo",
		"user/" + strings.Repeat("a", 65) + "/foo",
		"user/tap/" + strings.Repeat("a", 129),
	}
	for _, v := range invalid {
		assert.Error(t, validateFormulaArg(v), "%q should be rejected", v)
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

func TestCheckCmd_SkipNotInstalledNeedsMinVersion(t *testing.T) {
	// A bad flag combination is a configuration error. Under
	// --munki-installcheck it must not read as "nothing to do" (1).
	tests := []struct {
		args     []string
		expected int
	}{
		{[]string{"check", "--skip-not-installed", "--munki-installcheck", "git"}, 0},
		{[]string{"check", "--skip-not-installed", "git"}, 1},
	}
	for _, tt := range tests {
		cmd := buildRootCmd(newMockConfig(), "test")
		cmd.SetArgs(tt.args)
		err := cmd.Execute()
		require.Error(t, err)
		assert.Equal(t, tt.expected, exitCodeFor(err), "args %v", tt.args)
	}
}

func TestCheckCmd_InvalidFormulaNameExitCode(t *testing.T) {
	tests := []struct {
		args     []string
		expected int
	}{
		{[]string{"check", "--munki-installcheck", "../git"}, 0},
		{[]string{"check", "../git"}, 1},
	}
	for _, tt := range tests {
		cmd := buildRootCmd(newMockConfig(), "test")
		cmd.SetArgs(tt.args)
		err := cmd.Execute()
		require.Error(t, err)
		assert.Equal(t, tt.expected, exitCodeFor(err), "args %v", tt.args)
	}
}

func TestCheckCmd_CheckErrorExitCode(t *testing.T) {
	// The mock executor returns no output, so the brew info JSON cannot be
	// parsed. Any runtime error takes the same exit-code path.
	tests := []struct {
		args     []string
		expected int
	}{
		// Munki reads 0 as "install needed". An error must not read as compliant.
		{[]string{"check", "--no-cache", "--munki-installcheck", "git"}, 0},
		{[]string{"check", "--no-cache", "git"}, 1},
	}
	for _, tt := range tests {
		cmd := buildRootCmd(newMockConfig(), "test")
		cmd.SetArgs(tt.args)
		err := cmd.Execute()
		require.Error(t, err)
		assert.Equal(t, tt.expected, exitCodeFor(err), "args %v", tt.args)
	}
}

func TestExitCodeFor(t *testing.T) {
	assert.Equal(t, 1, exitCodeFor(errors.New("plain")))
	assert.Equal(t, 0, exitCodeFor(&exitCodeError{code: 0, err: errors.New("x")}))
	assert.Equal(t, 0, exitCodeFor(fmt.Errorf("wrapped: %w", &exitCodeError{code: 0, err: errors.New("x")})))
}

func TestCheckCmd_InvalidMinVersionIsUsageError(t *testing.T) {
	var out bytes.Buffer
	cmd := buildRootCmd(newMockConfig(), "test")
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"check", "--min-version=v2.x", "--skip-not-installed", "--munki-installcheck", "git"})
	err := cmd.Execute()
	require.Error(t, err)
	// A typo in a pkginfo must make the Munki item fail, not read as compliant.
	assert.Equal(t, 0, exitCodeFor(err))
	// Usage help is still shown for argument errors.
	assert.Contains(t, out.String(), "Usage:")
}

func TestCheckCmd_RuntimeErrorHasNoUsage(t *testing.T) {
	var out bytes.Buffer
	cmd := buildRootCmd(newMockConfig(), "test")
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"check", "--no-cache", "--munki-installcheck", "git"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.NotContains(t, out.String(), "Usage:")
	// main() prints the error; cobra must not print it a second time.
	assert.NotContains(t, out.String(), "Error:")
}
