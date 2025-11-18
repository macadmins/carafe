package testshell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sync"

	"github.com/macadmins/carafe/shell"
	"github.com/pkg/errors"
)

type capturingExecutor struct {
	mu            sync.Mutex
	captured      *[]*shell.Cmd
	inputs        *[]io.Reader
	stdout        []string
	stderr        []string
	mappedStdouts map[*regexp.Regexp]string
	mappedStderrs map[*regexp.Regexp]string
	pid           int
	err           error
	errIdx        int
	idx           int
}

// CapturingExecutor returns an executor that simply returns the commands that were run.
func CapturingExecutor() (shell.Executor, *[]*shell.Cmd) {
	captured := new([]*shell.Cmd)
	executor := NewExecutor(CaptureTo(captured))
	return executor, captured
}

// OutputExecutor returns an executor with the given outputs.
func OutputExecutor(outputs ...string) shell.Executor {
	return NewExecutor(WithStdout(outputs...))
}

func (c *capturingExecutor) Run(cmd *shell.Cmd) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.captured != nil {
		*c.captured = append(*c.captured, cmd)
	}

	if c.inputs != nil {
		native := (*exec.Cmd)(cmd)
		*c.inputs = append(*c.inputs, native.Stdin)
	}

	for regex, stdout := range c.mappedStdouts {
		if regex.MatchString(cmd.String()) {
			_, _ = fmt.Fprint(cmd.Stdout, stdout)
			return c.maybeReturnError()
		}
	}

	if c.stdout != nil {
		if c.idx >= len(c.stdout) {
			return errors.New("mock executor did not have enough registered stdouts")
		}

		if cmd.Stdout != nil {
			_, _ = fmt.Fprintf(cmd.Stdout, "%s", c.stdout[c.idx])
		}
	}

	if c.pid != 0 {
		cmd.Process = &os.Process{Pid: c.pid}
	}

	for regex, stderr := range c.mappedStderrs {
		if regex.MatchString(cmd.String()) {
			_, _ = fmt.Fprint(cmd.Stderr, stderr)
			return c.maybeReturnError()
		}
	}

	if c.stderr != nil {
		if c.idx >= len(c.stderr) {
			return errors.New("mock executor did not have enough registered stderrs")
		}

		if cmd.Stderr != nil {
			_, _ = fmt.Fprintf(cmd.Stderr, "%s", c.stderr[c.idx])
		}
	}

	return c.maybeReturnError()
}

func (c *capturingExecutor) maybeReturnError() error {
	c.idx++

	if c.errIdx == 0 || (c.errIdx > 0 && c.idx == c.errIdx) {
		return c.err
	}

	return nil
}

func (c *capturingExecutor) Start(cmd *shell.Cmd) error {
	return c.Run(cmd)
}

func (c *capturingExecutor) RunCancellable(_ context.Context, cmd *shell.Cmd) error {
	return c.Run(cmd)
}

// NewExecutor creates a mock Executor with the given options.
func NewExecutor(opts ...ExecutorOption) shell.Executor {
	executor := &capturingExecutor{}
	for _, opt := range opts {
		opt(executor)
	}

	return executor
}

// NullConsole returns a Console that doesn't do anything.
func NullConsole() shell.Console {
	return shell.Console{
		Out: io.Discard,
		Err: io.Discard,
		In:  new(bytes.Buffer),
	}
}
