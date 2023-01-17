package cmd

import "strings"

type ShellCommand []string

func (c ShellCommand) String() string {
	return strings.Join(c, " ")
}
