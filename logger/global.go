package logger

import "gopkg.in/go-mixed/go-common.v1/utils"

var globalLogger *Logger

func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

func GetGlobalLogger() *Logger {
	return globalLogger
}

// BuildGlobalLogger 新建全局用的Logger
// errorFilename 非空时表示将ERROR以上的日志写入到此文件中
func BuildGlobalLogger(filename string, errorFilename string) (*Logger, error) {
	logger, err := NewLogger(LoggerOptions{
		FilePath:      filename,
		ErrorFilePath: errorFilename,
	})

	if err != nil {
		return nil, err
	}
	SetGlobalLogger(logger)
	utils.SetGlobalILogger(logger.ToILogger())
	return logger, nil
}
