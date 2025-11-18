package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"

	"github.com/gonuts/go-shellquote"
)

// Cmd is a type alias for exec.Cmd without the runnable attached methods.
// Running this command can be done through an Executor.
type Cmd exec.Cmd

// Console is a convenience grouping for standard file descriptors.
type Console struct {
	Out io.Writer
	Err io.Writer
	In  io.Reader
}

// Format formats the command, which adheres to the Formatter interface.
// '%s' in a format string will return the command in a form suitable for copy/pasting into a terminal.
// '%v' will return the command (surrounded by backticks) with information about the working directory.
func (c *Cmd) Format(state fmt.State, verb rune) {
	switch verb {
	case 's':
		_, _ = fmt.Fprint(state, c.String())
	case 'v':
		_, _ = fmt.Fprint(state, "`")
		_, _ = fmt.Fprint(state, c.String())
		_, _ = fmt.Fprintf(state, "`")
		if c.Dir != "" {
			_, _ = fmt.Fprintf(state, " in %s", c.Dir)
		}
	}
}

// NewCommand returns a Cmd to run the command with the given args. See exec.Command.
func NewCommand(name string, arg ...string) *Cmd {
	return (*Cmd)(exec.Command(name, arg...)) // #nosec G204
}

// NewCommandWithContext returns a Cmd to run the command with the given args, using the provided context. See
// exec.CommandContext.
func NewCommandWithContext(ctx context.Context, name string, arg ...string) *Cmd {
	return (*Cmd)(exec.CommandContext(ctx, name, arg...)) // #nosec G204
}

// String returns a representation of the command suitable for copy/pasting into a shell.
func (c *Cmd) String() string {
	var cmdParts []string

	// only add the env vars that differ from the current environment
	// to prevent dumping lots of unnecessary/sensitive environment variables
	cmdParts = append(cmdParts, newEnvVars(c.Env)...)
	cmdParts = append(cmdParts, c.Args...)
	return shellquote.Join(cmdParts...)
}

func newEnvVars(envs []string) []string {
	if len(envs) == 0 {
		return []string{}
	}

	osEnv := os.Environ()
	slices.Sort(osEnv)

	var newEnvs []string
	for _, a := range envs {
		if _, ok := slices.BinarySearch(osEnv, a); !ok {
			newEnvs = append(newEnvs, a)
		}
	}

	return newEnvs
}

// SetConsole sets the standard file descriptors for the command.
func (c *Cmd) SetConsole(console Console) {
	c.Stdout = console.Out
	c.Stderr = console.Err
	c.Stdin = console.In
}

func (c *Cmd) Native() *exec.Cmd {
	return (*exec.Cmd)(c)
}
