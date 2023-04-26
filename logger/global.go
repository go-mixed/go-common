package logger

import "gopkg.in/go-mixed/go-common.v1/utils"

var globalLogger *Logger

func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

func GetGlobalLogger() *Logger {
	return globalLogger
}

func GetILogger() utils.ILogger {
	if globalLogger != nil {
		return globalLogger.ILogger()
	}
	return utils.NewDefaultLogger()
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
	return logger, nil
}