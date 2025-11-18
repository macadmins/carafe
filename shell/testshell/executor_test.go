package testshell

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"github.com/macadmins/carafe/shell"
)

func TestCapturingExecutor_Run_CaptureTo(t *testing.T) {
	captured := []*shell.Cmd{}
	cmd := shell.NewCommand("command-that-does-not-exist")
	err := NewExecutor(CaptureTo(&captured)).Run(cmd)
	require.NoError(t, err)
	assert.Len(t, captured, 1)
	assert.Equal(t, cmd, captured[0])
}

func TestCapturingExecutor_Run_AlwaysError(t *testing.T) {
	captured := []*shell.Cmd{}
	cmd := shell.NewCommand("command-that-does-not-exist")
	expectedErr := errors.New("exec err")
	err := NewExecutor(CaptureTo(&captured), AlwaysError(expectedErr)).Run(cmd)
	assert.Len(t, captured, 1)
	assert.Equal(t, expectedErr, err)
}

func TestCapturingExecutor_Run_AlwaysErrorOnCommandIndex(t *testing.T) {
	captured := []*shell.Cmd{}
	goodCmd := shell.NewCommand("good-command")
	badCmd := shell.NewCommand("bad-command")
	expectedErr := errors.New("exec err")
	executor := NewExecutor(CaptureTo(&captured), AlwaysErrorOnCommandIndex(expectedErr, 3))

	err1 := executor.Run(goodCmd)
	err2 := executor.Run(goodCmd)
	err3 := executor.Run(badCmd)

	assert.Len(t, captured, 3)
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, expectedErr, err3)
}

func TestCapturingExecutor_Run_MultipleWithStdout(t *testing.T) {
	captured := []*shell.Cmd{}
	buffs := make([]bytes.Buffer, 3)
	executor := NewExecutor(CaptureTo(&captured), WithStdout("a", "b", "c"))

	for i := 0; i < 3; i++ {
		cmd := shell.NewCommand("command-that-does-not-exist")
		cmd.Stdout = &buffs[i]
		err := executor.Run(cmd)
		require.NoError(t, err)
	}

	assert.Len(t, captured, 3)
	assert.Equal(t, "a", buffs[0].String())
	assert.Equal(t, "b", buffs[1].String())
	assert.Equal(t, "c", buffs[2].String())
}

func TestCapturingExecutor_Run_MultipleWithStdout_TooFew(t *testing.T) {
	executor := NewExecutor(WithStdout("a"))

	cmd := shell.NewCommand("command-that-does-not-exist")
	err := executor.Run(cmd)
	require.NoError(t, err)
	err = executor.Run(cmd)
	assert.Error(t, err)
}

func TestCapturingExecutor_Run_WithStderr(t *testing.T) {
	captured := []*shell.Cmd{}
	buffs := make([]bytes.Buffer, 3)
	executor := NewExecutor(CaptureTo(&captured), WithStderr("a", "b", "c"))

	for i := 0; i < 3; i++ {
		cmd := shell.NewCommand("command-that-does-not-exist")
		cmd.Stderr = &buffs[i]
		err := executor.Run(cmd)
		require.NoError(t, err)
	}

	assert.Len(t, captured, 3)
	assert.Equal(t, "a", buffs[0].String())
	assert.Equal(t, "b", buffs[1].String())
	assert.Equal(t, "c", buffs[2].String())
}
