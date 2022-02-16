package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

// Command represents a single command which can be executed
type Command struct {
	Path string
	Args []string

	Env          []string
	Timeout      time.Duration
	StderrWriter io.Writer
	StdoutWriter io.Writer
	WorkingDir   string
	executed     bool
	exitCode     int
	// stderr and stdout retrieve the output after the command was executed
	stderr   bytes.Buffer
	stdout   bytes.Buffer
	combined bytes.Buffer
}

// EnvVars represents a map where the key is the name of the env variable
// and the value is the value of the variable
//
// Example:
//
//  env := map[string]string{"ENV": "VALUE"}
//
type EnvVars map[string]string

// NewCommand creates a new command
// You can add option with variadic option argument
// Default timeout is set to 30 minutes
//
// Example:
//      c := cmd.NewCommand("echo hello", function (c *Command) {
//		    c.WorkingDir = "/tmp"
//      })
//      c.Execute()
//
// or you can use existing options functions
//
//      c := cmd.NewCommand("echo hello", cmd.WithStandardStreams)
//      c.Execute()
//
func NewCommand(path string, args []string, options ...func(*Command)) *Command {
	c := &Command{
		Path:     path,
		Args:     args,
		Timeout:  30 * time.Minute,
		executed: false,
		Env:      []string{},
	}

	if filepath.Base(c.Path) == c.Path {
		if lp, err := exec.LookPath(c.Path); err == nil {
			c.Path = lp
		}
	}

	c.StdoutWriter = io.MultiWriter(&c.stdout, &c.combined)
	c.StderrWriter = io.MultiWriter(&c.stderr, &c.combined)

	for _, o := range options {
		o(c)
	}

	return c
}

// WithStandardStreams is used as an option by the NewCommand constructor function and writes the output streams
// to stderr and stdout of the operating system
//
// Example:
//
//     c := cmd.NewCommand("echo hello", cmd.WithStandardStreams)
//     c.Execute()
//
func WithStandardStreams(c *Command) {
	c.StdoutWriter = io.MultiWriter(os.Stdout, &c.stdout, &c.combined)
	c.StderrWriter = io.MultiWriter(os.Stderr, &c.stdout, &c.combined)
}

// WithCustomStdout allows to add custom writers to stdout
func WithCustomStdout(writers ...io.Writer) func(c *Command) {
	return func(c *Command) {
		writers = append(writers, &c.stdout, &c.combined)
		c.StdoutWriter = io.MultiWriter(writers...)
	}
}

// WithCustomStderr allows to add custom writers to stderr
func WithCustomStderr(writers ...io.Writer) func(c *Command) {
	return func(c *Command) {
		writers = append(writers, &c.stderr, &c.combined)
		c.StderrWriter = io.MultiWriter(writers...)
	}
}

// WithTimeout sets the timeout of the command
//
// Example:
//     cmd.NewCommand("sleep 10;", cmd.WithTimeout(500))
//
func WithTimeout(t time.Duration) func(c *Command) {
	return func(c *Command) {
		c.Timeout = t
	}
}

// WithoutTimeout disables the timeout for the command
func WithoutTimeout(c *Command) {
	c.Timeout = 0
}

// WithWorkingDir sets the current working directory
func WithWorkingDir(dir string) func(c *Command) {
	return func(c *Command) {
		c.WorkingDir = dir
	}
}

// WithInheritedEnvironment uses the env from the current process and
// allow to add more variables.
func WithInheritedEnvironment(env EnvVars) func(c *Command) {
	return func(c *Command) {
		c.Env = os.Environ()

		// Set custom variables
		fn := WithEnvironmentVariables(env)
		fn(c)
	}
}

// WithEnvironmentVariables sets environment variables for the executed command
func WithEnvironmentVariables(env EnvVars) func(c *Command) {
	return func(c *Command) {
		for key, value := range env {
			c.AddEnv(key, value)
		}
	}
}

func (c *Command) IsExecutable() bool {
	fileInfo, err := os.Stat(c.Path)
	if err != nil || fileInfo.IsDir() {
		return false
	}

	if runtime.GOOS == "windows" {
		return true
	}

	if fileInfo.Mode()&0111 != 0 {
		return true
	}

	return false
}

// AddEnv adds an environment variable to the command
// If a variable gets passed like ${VAR_NAME} the env variable will be read out by the current shell
func (c *Command) AddEnv(key string, value string) {
	value = os.ExpandEnv(value)
	c.Env = append(c.Env, fmt.Sprintf("%s=%s", key, value))
}

// Stdout returns the output to stdout
func (c *Command) Stdout() string {
	c.isExecuted("Stdout")
	return c.stdout.String()
}

// Stderr returns the output to stderr
func (c *Command) Stderr() string {
	c.isExecuted("Stderr")
	return c.stderr.String()
}

// Combined returns the combined output of stderr and stdout according to their timeline
func (c *Command) Combined() string {
	c.isExecuted("Combined")
	return c.combined.String()
}

//ExitCode returns the exit code of the command
func (c *Command) ExitCode() int {
	c.isExecuted("ExitCode")
	return c.exitCode
}

//Executed returns if the command was already executed
func (c *Command) Executed() bool {
	return c.executed
}

func (c *Command) isExecuted(property string) {
	if !c.executed {
		panic("Can not read " + property + " if command was not executed.")
	}
}

// Execute executes the command and writes the results into it's own instance
// The results can be received with the Stdout(), Stderr() and ExitCode() methods
func (c *Command) Execute() error {
	return c.ExecuteContext(context.Background())
}

// ExecuteContext executes the command with a context.Context, and writes the results into it's own instance
// The results can be received with the Stdout(), Stderr() and ExitCode() methods
//
// You can stop the running command via Context's cancel
func (c *Command) ExecuteContext(ctx context.Context) error {

	// replace to timeout context
	if c.Timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	cmd := createBaseCommand(c, ctx)
	cmd.Env = c.Env
	cmd.Stdout = c.StdoutWriter
	cmd.Stderr = c.StderrWriter
	cmd.Dir = c.WorkingDir

	var err error
	if err = cmd.Start(); err != nil {
		return err
	}

	// 由于可能会出现无法Kill进程的情况, 将Wait放入协程中运行
	var errChan = make(chan error)
	go func() {
		err1 := cmd.Wait()
		errChan <- err1
	}()

	select {
	case <-ctx.Done(): // 监听ctx退出, 并检查是否是超时退出的情况
		if ctxErr := ctx.Err(); ctxErr != nil {
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				err = fmt.Errorf("command timed out after %v", c.Timeout)
				// fmt.Errorf("Timeout occurred and can not kill process with pid %v", cmd.Process.Pid)
			}
		}
	case err = <-errChan: // 正常运行结束
		c.exitCode = 0
		if err != nil {
			c.getExitCode(err)
		}
	}

	//return
	c.executed = true

	return err
}

func (c *Command) getExitCode(err error) {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			c.exitCode = status.ExitStatus()
		}
	}
}

func (c *Command) String() string {
	cmd := createBaseCommand(c, context.TODO())
	return cmd.String()
}
