package cmd

func getCommandPrefix(c *Command) ShellCommand {
	if c.privilegedInDocker {
		return ShellCommand{"nsenter", "-t", "1", "-m", "-u", "-n", "-i"}
	} else if !c.IsExecutable() {
		return ShellCommand{"/bin/sh", "-c"}
	}

	return nil
}
