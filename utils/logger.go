package utils

import (
	"github.com/utahta/go-cronowriter"
	"go-common/utils/io"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var logger *zap.Logger
var sugarLogger *zap.SugaredLogger

const (
	DEBUG = iota
	INFO
	WARN
	PANIC
	ERROR
	FATAL
)

type ILogger interface {
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Panic(v ...interface{})
	Panicf(format string, v ...interface{})

	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Warn(v ...interface{})
	Warnf(format string, v ...interface{})
}

type DefaultLogger struct {
	stdOutLog *log.Logger
	stdErrLog *log.Logger
	Level     int
}

func NewDefaultLogger() ILogger {
	return &DefaultLogger{
		stdOutLog: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile|log.Lmsgprefix),
		stdErrLog: log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile|log.Lmsgprefix),
	}
}

// InitLogger 初始化Logger
// errorFilename 传递非空字符串，表示将错误分开写入到此文件中
func InitLogger(filename string, errorFilename string) {
	// 创建文件夹
	_ = io_utils.MustMkdirAll(filepath.Dir(filename), os.ModePerm)
	if errorFilename != "" {
		_ = io_utils.MustMkdirAll(filepath.Dir(errorFilename), os.ModePerm)
	}

	// 获取log writer
	writeSyncer := getLogWriter(filename)
	writeStdout := zapcore.AddSync(os.Stdout)
	encoder := getLogEncoder()

	errorLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zapcore.ErrorLevel
	})
	// 默认最低显示debug
	minLevel := zapcore.DebugLevel
	logEnvLevel := strings.ToLower(os.Getenv("ZAP_LOG_LEVEL"))
	if logEnvLevel == "info" {
		minLevel = zapcore.InfoLevel
	} else if logEnvLevel == "warn" {
		minLevel = zapcore.WarnLevel
	}
	// 错误log和运行log存储在一起
	if errorFilename == "" {
		normalLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level >= minLevel
		})
		core := zapcore.NewTee(
			zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writeSyncer, writeStdout), normalLevel),
		)
		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(errorLevel))
	} else {
		normalLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level >= minLevel && level < zapcore.ErrorLevel
		})
		// 分开存储
		errorWriteSyncer := getLogWriter(errorFilename)
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

func GetSugaredLogger() ILogger {
	return sugarLogger
}

func getLogEncoder() zapcore.Encoder {
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

func (d DefaultLogger) Fatal(v ...interface{}) {
	if d.Level <= FATAL {
		d.stdErrLog.SetPrefix("FATAL\t")
		d.stdErrLog.Fatal(v...)
	}
}

func (d DefaultLogger) Fatalf(format string, v ...interface{}) {
	if d.Level <= FATAL {
		d.stdErrLog.SetPrefix("FATAL\t")
		d.stdErrLog.Fatalf(format, v...)
	}
}

func (d DefaultLogger) Error(v ...interface{}) {
	if d.Level <= ERROR {
		d.stdErrLog.SetPrefix("ERROR\t")
		d.stdErrLog.Print(v...)
	}
}

func (d DefaultLogger) Errorf(format string, v ...interface{}) {
	if d.Level <= ERROR {
		d.stdErrLog.SetPrefix("ERROR\t")
		d.stdErrLog.Printf(format, v...)
	}
}

func (d DefaultLogger) Panic(v ...interface{}) {
	if d.Level <= PANIC {
		d.stdErrLog.SetPrefix("PANIC\t")
		d.stdErrLog.Panic(v...)
	}
}

func (d DefaultLogger) Panicf(format string, v ...interface{}) {
	if d.Level <= PANIC {
		d.stdErrLog.SetPrefix("PANIC\t")
		d.stdErrLog.Panicf(format, v...)
	}
}

func (d DefaultLogger) Debug(v ...interface{}) {
	if d.Level <= DEBUG {
		d.stdOutLog.SetPrefix("DEBUG\t")
		d.stdOutLog.Print(v...)
	}
}

func (d DefaultLogger) Debugf(format string, v ...interface{}) {
	if d.Level <= DEBUG {
		d.stdOutLog.SetPrefix("DEBUG\t")
		d.stdOutLog.Printf(format, v...)
	}
}

func (d DefaultLogger) Info(v ...interface{}) {
	if d.Level <= INFO {
		d.stdOutLog.SetPrefix("INFO\t")
		d.stdOutLog.Print(v...)
	}
}

func (d DefaultLogger) Infof(format string, v ...interface{}) {
	if d.Level <= INFO {
		d.stdOutLog.SetPrefix("INFO\t")
		d.stdOutLog.Printf(format, v...)
	}
}

func (d DefaultLogger) Warn(v ...interface{}) {
	if d.Level <= WARN {
		d.stdOutLog.SetPrefix("WARN\t")
		d.stdOutLog.Print(v...)
	}
}

func (d DefaultLogger) Warnf(format string, v ...interface{}) {
	if d.Level <= WARN {
		d.stdOutLog.SetPrefix("WARN\t")
		d.stdOutLog.Printf(format, v...)
	}
}
