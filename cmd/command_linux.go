package cmd

import (
	"context"
	"os/exec"
)

func createBaseCommand(c *Command, ctx context.Context) *exec.Cmd {
	if !c.IsExecutable() {
		var args []string
		args = append(args, "-c", c.Path)
		args = append(args, c.Args...)
		return exec.CommandContext(ctx, "/bin/sh", args...)
	} else {
		return exec.CommandContext(ctx, c.Path, c.Args...)
	}
}
