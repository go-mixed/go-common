package cmd

func getCommandPrefix(c *Command) ShellCommand {
	if !c.IsExecutable() {
		return ShellCommand{"C:\\windows\\system32\\cmd.exe", "/C"}
	}

	return nil
}
