package main

import (
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
