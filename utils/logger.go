package utils

import (
	"github.com/utahta/go-cronowriter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	DEBUG = iota
	INFO
	WARN
	PANIC
	ERROR
	FATAL
)

type ILogger interface {
	Fatal(v ...any)
	Fatalf(format string, v ...any)
	Error(v ...any)
	Errorf(format string, v ...any)
	Panic(v ...any)
	Panicf(format string, v ...any)

	Debug(v ...any)
	Debugf(format string, v ...any)
	Info(v ...any)
	Infof(format string, v ...any)
	Warn(v ...any)
	Warnf(format string, v ...any)
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

const ZapConsoleLogLevel = "ZAP_CONSOLE_LOG_LEVEL"
const ZapLogLevel = "ZAP_LOG_LEVEL"
const ZapLogEncoder = "ZAP_LOG_ENCODER"

var globalLogger *zap.Logger

func SetGlobalLogger(logger *zap.Logger) {
	globalLogger = logger
}

func GetGlobalLogger() *zap.Logger {
	return globalLogger
}

func GetILogger() ILogger {
	if globalLogger != nil {
		return globalLogger.Sugar()
	}
	return NewDefaultLogger()
}

// InitLogger 初始化Logger
// errorFilename 传递非空字符串，表示将错误分开写入到此文件中
func InitLogger(filename string, errorFilename string) (*zap.Logger, error) {
	// 创建文件夹
	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return nil, err
	}
	if errorFilename != "" {
		if err := os.MkdirAll(filepath.Dir(errorFilename), os.ModePerm); err != nil {
			return nil, err
		}
	}

	var logger *zap.Logger

	// 获取log writer
	writeSyncer := getLogWriter(filename)
	writeStdout := zapcore.AddSync(os.Stdout)
	consoleEncoder := getConsoleEncoder()
	encoder := getLogEncoder()

	errorLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zapcore.ErrorLevel
	})

	consoleLevel := getLevel(ZapConsoleLogLevel, true)

	// 错误log和运行log位于一个文件
	if errorFilename == "" {
		level := getLevel(ZapLogLevel, true)

		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, writeStdout, consoleLevel), // 控制台输出
			zapcore.NewCore(encoder, writeSyncer, level),               // 文件输出
		)
		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(errorLevel))
	} else { // 分开写入
		level := getLevel(ZapLogLevel, false)

		errorWriteSyncer := getLogWriter(errorFilename)
		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, writeStdout, consoleLevel), // 控制台输出
			zapcore.NewCore(encoder, writeSyncer, level),               // 非异常日志的文件输出
			zapcore.NewCore(encoder, errorWriteSyncer, errorLevel),     // 异常日志的文件输出
		)
		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(errorLevel))
	}

	SetGlobalLogger(logger)
	return logger, nil
}

func getConsoleEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	switch strings.ToLower(os.Getenv(ZapLogEncoder)) {
	case "json":
		return zapcore.NewJSONEncoder(encoderConfig)
	default:
		return zapcore.NewConsoleEncoder(encoderConfig)
	}
}

// "ZAP_LOG_LEVEL" or "ZAP_CONSOLE_LOG_LEVEL"
func getLevel(env string, withError bool) zap.LevelEnablerFunc {
	minLevel := zapcore.DebugLevel
	logEnvLevel := strings.ToLower(os.Getenv(env))
	if logEnvLevel == "disabled" {
		return func(level zapcore.Level) bool {
			return false
		}
	} else if logEnvLevel == "info" {
		minLevel = zapcore.InfoLevel
	} else if logEnvLevel == "warn" {
		minLevel = zapcore.WarnLevel
	}

	if withError {
		return func(level zapcore.Level) bool {
			return level >= minLevel
		}
	} else {
		return func(level zapcore.Level) bool {
			return level >= minLevel && level < zapcore.ErrorLevel
		}
	}
}

func getLogWriter(filename string) zapcore.WriteSyncer {
	ext := filepath.Ext(filename)
	path := filename[0:len(filename)-len(ext)] + ".%Y-%m-%d" + ext

	return zapcore.AddSync(cronowriter.MustNew(path))
}

func (d DefaultLogger) Fatal(v ...any) {
	if d.Level <= FATAL {
		d.stdErrLog.SetPrefix("FATAL\t")
		d.stdErrLog.Fatal(v...)
	}
}

func (d DefaultLogger) Fatalf(format string, v ...any) {
	if d.Level <= FATAL {
		d.stdErrLog.SetPrefix("FATAL\t")
		d.stdErrLog.Fatalf(format, v...)
	}
}

func (d DefaultLogger) Error(v ...any) {
	if d.Level <= ERROR {
		d.stdErrLog.SetPrefix("ERROR\t")
		d.stdErrLog.Print(v...)
	}
}

func (d DefaultLogger) Errorf(format string, v ...any) {
	if d.Level <= ERROR {
		d.stdErrLog.SetPrefix("ERROR\t")
		d.stdErrLog.Printf(format, v...)
	}
}

func (d DefaultLogger) Panic(v ...any) {
	if d.Level <= PANIC {
		d.stdErrLog.SetPrefix("PANIC\t")
		d.stdErrLog.Panic(v...)
	}
}

func (d DefaultLogger) Panicf(format string, v ...any) {
	if d.Level <= PANIC {
		d.stdErrLog.SetPrefix("PANIC\t")
		d.stdErrLog.Panicf(format, v...)
	}
}

func (d DefaultLogger) Debug(v ...any) {
	if d.Level <= DEBUG {
		d.stdOutLog.SetPrefix("DEBUG\t")
		d.stdOutLog.Print(v...)
	}
}

func (d DefaultLogger) Debugf(format string, v ...any) {
	if d.Level <= DEBUG {
		d.stdOutLog.SetPrefix("DEBUG\t")
		d.stdOutLog.Printf(format, v...)
	}
}

func (d DefaultLogger) Info(v ...any) {
	if d.Level <= INFO {
		d.stdOutLog.SetPrefix("INFO\t")
		d.stdOutLog.Print(v...)
	}
}

func (d DefaultLogger) Infof(format string, v ...any) {
	if d.Level <= INFO {
		d.stdOutLog.SetPrefix("INFO\t")
		d.stdOutLog.Printf(format, v...)
	}
}

func (d DefaultLogger) Warn(v ...any) {
	if d.Level <= WARN {
		d.stdOutLog.SetPrefix("WARN\t")
		d.stdOutLog.Print(v...)
	}
}

func (d DefaultLogger) Warnf(format string, v ...any) {
	if d.Level <= WARN {
		d.stdOutLog.SetPrefix("WARN\t")
		d.stdOutLog.Printf(format, v...)
	}
}
