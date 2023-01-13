package cmd

import (
	"context"
	"os/exec"
)

func getCommandPrefix(c *Command) ShellCommand {
	if !c.IsExecutable() {
		return ShellCommand{"C:\\windows\\system32\\cmd.exe", "/C"}
	}

	return nil
}

func (c *Command) buildCommandContext(ctx context.Context) *exec.Cmd {
	prefix := getCommandPrefix(c)
	var shell ShellCommand
	// windows 使用shellwords.Parse会删除路径中的\，故保持原样
	shell = append(prefix, c.Path)
	// 添加原args到指令末尾
	shell = append(shell, c.Args...)

	return exec.CommandContext(ctx, shell[0], shell[1:]...)
}
