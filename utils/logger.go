package utils

import (
	"github.com/utahta/go-cronowriter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
)

var logger *zap.Logger
var sugarLogger *zap.SugaredLogger

/**
 * 初始化Logger
 * errorFilename 传递非空字符串，表示将错误分开写入到此文件中
 */
func InitLogger(filename string, errorFilename string) {
	// 创建文件夹
	_ = os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	_ = os.MkdirAll(filepath.Dir(errorFilename), os.ModePerm)

	// 获取log writer
	writeSyncer := getLogWriter(filename)
	writeStdout := zapcore.AddSync(os.Stdout)
	encoder := getEncoder()

	errorLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zapcore.ErrorLevel
	})

	// 错误log和运行log存储在一起
	if errorFilename == "" {
		normalLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return true
		})

		core := zapcore.NewTee(
			zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writeSyncer, writeStdout), normalLevel),
		)

		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(errorLevel))
	} else { // 分开存储
		errorWriteSyncer := getLogWriter(errorFilename)

		normalLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level >= zapcore.DebugLevel && level < zapcore.ErrorLevel
		})

		core := zapcore.NewTee(
			zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writeSyncer, writeStdout), normalLevel),
			zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(errorWriteSyncer, writeStdout), errorLevel),
		)

		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(errorLevel))
	}

	sugarLogger = logger.Sugar()
}

func GetLogger() *zap.Logger {
	return logger
}

func GetSugaredLogger() *zap.SugaredLogger {
	return sugarLogger
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	if true {
		return zapcore.NewConsoleEncoder(encoderConfig)
	}
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getLogWriter(filename string) zapcore.WriteSyncer {
	ext := filepath.Ext(filename)
	path := filename[0:len(filename)-len(ext)] + ".%Y-%m-%d" + ext

	return zapcore.AddSync(cronowriter.MustNew(path))
}
