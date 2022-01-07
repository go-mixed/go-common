package cmd

import (
	"context"
	"os/exec"
)

func createBaseCommand(c *Command, ctx context.Context) *exec.Cmd {
	if c.useShellPrefix {
		var args []string
		args = append(args, "/C", c.Path)
		args = append(args, c.Args...)
		return exec.CommandContext(ctx, `C:\windows\system32\cmd.exe`, args...)
	} else {
		return exec.CommandContext(ctx, c.Path, c.Args...)
	}
}
