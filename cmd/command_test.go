package cmd

import (
	"bytes"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
	"testing"
	"time"
)

func gbkToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

func TestCommand(t *testing.T) {
	t.Logf("test echo")
	command := NewCommand("echo", []string{"123"})
	t.Logf("command: %s", command.String())
	if err := command.Execute(); err != nil {
		t.Errorf("%s", err.Error())
	}

	t.Logf("stdout: %s", command.Stdout())
	t.Logf("exit code: %d", command.ExitCode())
}

func TestCommandTimeout(t *testing.T) {
	t.Logf("test timeout")

	now := time.Now()
	command := NewCommand("ping", []string{"127.0.0.1", "-n", "4", ">", "nul"}, WithTimeout(500*time.Millisecond))
	t.Logf("command: %s", command.String())
	if err := command.Execute(); err != nil {
		t.Errorf("%s", err.Error())
	}

	t.Logf("durations: %0.3f", time.Since(now).Seconds())

	s, _ := gbkToUtf8([]byte(command.Stdout()))

	t.Logf("stdout: %s", s)
	t.Logf("stderr: %s", command.Stderr())
	t.Logf("exit code: %d", command.ExitCode())
}
