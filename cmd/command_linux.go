package cmd

import (
	"context"
	"github.com/mattn/go-shellwords"
	"os/exec"
)

func getCommandPrefix(c *Command) ShellCommand {
	if c.privilegedInDocker {
		return ShellCommand{"nsenter", "-t", "1", "-m", "-u", "-n", "-i"}
	} else if !c.IsExecutable() {
		return ShellCommand{"/bin/sh", "-c"}
	}

	return nil
}

func (c *Command) buildCommandContext(ctx context.Context) *exec.Cmd {
	prefix := getCommandPrefix(c)
	var shell ShellCommand
	// 將原c.Path字符串解析為ShellCommand
	if newCommand, err := shellwords.Parse(c.Path); err != nil {
		shell = append(prefix, c.Path) // 無法解析，保持原樣
	} else {
		shell = append(prefix, newCommand...)
	}
	// 添加原args到指令末尾
	shell = append(shell, c.Args...)

	return exec.CommandContext(ctx, shell[0], shell[1:]...)
}
