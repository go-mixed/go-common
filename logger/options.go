package logger

import (
	"gopkg.in/go-mixed/go-common.v1/utils/io"
	"path/filepath"
)

type LoggerOptions struct {
	FilePath      string `json:"file_path" yaml:"file_path" validate:"required"`
	ErrorFilePath string `json:"error_file_path" yaml:"error_file_path"`

	FileEncoder     string `json:"file_encoder" yaml:"file_encoder"`
	FileMinLevel    string `json:"file_level" yaml:"file_level"`
	ConsoleMinLevel string `json:"console_level" yaml:"console_level"`
}

func DefaultLoggerOptions() LoggerOptions {
	return LoggerOptions{
		FilePath:      filepath.Join(ioUtils.GetCurrentDir(), "logs", "app.log"),
		ErrorFilePath: "",

		FileEncoder:     "console",
		FileMinLevel:    "debug",
		ConsoleMinLevel: "debug",
	}
}
