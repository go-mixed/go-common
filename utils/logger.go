package utils

import (
	"github.com/utahta/go-cronowriter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapio"
	"io"
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

const ZapConsoleLevel = "ZAP_CONSOLE_LOG_LEVEL"
const ZapFileLevel = "ZAP_LOG_LEVEL"
const ZapFileEncoder = "ZAP_LOG_ENCODER"

type Logger struct {
	*zap.Logger

	FilePath      string
	ErrorFilePath string

	consoleLevel zapcore.Level
	fileLevel    zapcore.Level
	fileEncoder  *fileEncoder
}

var globalLogger *Logger

func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

func GetGlobalLogger() *Logger {
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
func InitLogger(filename string, errorFilename string) (*Logger, error) {
	// 创建文件夹
	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return nil, err
	}
	if errorFilename != "" {
		if err := os.MkdirAll(filepath.Dir(errorFilename), os.ModePerm); err != nil {
			return nil, err
		}
	}

	logger := &Logger{
		FilePath:      filename,
		ErrorFilePath: errorFilename,

		consoleLevel: getZapLevelFromEnv(ZapConsoleLevel),
		fileLevel:    getZapLevelFromEnv(ZapFileLevel),
		fileEncoder:  &fileEncoder{},
	}

	logger.buildFileEncoder(strings.ToLower(os.Getenv(ZapFileEncoder)))

	logger.Logger = zap.New(
		logger.buildCore(),
		zap.AddCaller(),
		zap.AddStacktrace(zap.LevelEnablerFunc(logger.errorLevelFunc)),
	)

	SetGlobalLogger(logger)
	return logger, nil
}

// SetFileLevel 修改文件输出的最低Level，可实时修改
func (l *Logger) SetFileLevel(level zapcore.Level) *Logger {
	l.fileLevel = level
	return l
}

// SetConsoleLevel 修改控制台输出的最低Level，可实时修改
func (l *Logger) SetConsoleLevel(level zapcore.Level) *Logger {
	l.consoleLevel = level
	return l
}

// SetFileEncoder 修改文件输出的encoder：console、json，可实时修改
func (l *Logger) SetFileEncoder(encoder string) *Logger {
	l.buildFileEncoder(encoder)
	return l
}

func (l *Logger) buildCore() zapcore.Core {

	errorLevelFunc := zap.LevelEnablerFunc(l.errorLevelFunc)
	var cores []zapcore.Core

	// 获取console的cores
	consoleEncoder := l.getConsoleEncoder()
	stdoutSyncer := zapcore.AddSync(os.Stdout)
	stderrSyncer := zapcore.AddSync(os.Stderr)
	cores = append(cores,
		zapcore.NewCore(consoleEncoder, stdoutSyncer, zap.LevelEnablerFunc(l.consoleLevelFunc)), // stdout输出
		zapcore.NewCore(consoleEncoder, stderrSyncer, errorLevelFunc),                           // stderr输出
	)

	// 获取文件的cores
	fileSyncer := l.getFileWriter(l.FilePath)
	cores = append(cores, zapcore.NewCore(l.fileEncoder, fileSyncer, zap.LevelEnablerFunc(l.fileLevelFunc))) // 常规信息的文件输出

	if l.ErrorFilePath != "" {
		errorSyncer := l.getFileWriter(l.ErrorFilePath)
		cores = append(cores, zapcore.NewCore(l.fileEncoder, errorSyncer, errorLevelFunc)) // 错误log单独输出
	} else {
		cores = append(cores, zapcore.NewCore(l.fileEncoder, fileSyncer, errorLevelFunc)) // 错误和常规的log在一个文件
	}

	return zapcore.NewTee(cores...)
}

func (l *Logger) getConsoleEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	return zapcore.NewConsoleEncoder(encoderConfig)
}

func (l *Logger) buildFileEncoder(encoder string) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	switch encoder {
	case "json", "JSON":
		l.fileEncoder.Encoder = zapcore.NewJSONEncoder(encoderConfig)
	default:
		l.fileEncoder.Encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}
}

func (l *Logger) errorLevelFunc(level zapcore.Level) bool {
	return level >= zap.ErrorLevel
}

func (l *Logger) fileLevelFunc(level zapcore.Level) bool {
	return level >= l.fileLevel && level < zap.ErrorLevel
}

func (l *Logger) consoleLevelFunc(level zapcore.Level) bool {
	return level >= l.consoleLevel && level < zap.ErrorLevel
}

func getZapLevelFromEnv(env string) zapcore.Level {
	minLevel := zapcore.DebugLevel
	envLevel := strings.ToLower(os.Getenv(env))
	if envLevel == "disabled" {
		return 0
	} else {
		if err := minLevel.UnmarshalText([]byte(envLevel)); err != nil {
			minLevel = zap.DebugLevel
		}
	}

	return minLevel
}

func (l *Logger) getFileWriter(filename string) zapcore.WriteSyncer {
	ext := filepath.Ext(filename)
	path := filename[0:len(filename)-len(ext)] + ".%Y-%m-%d" + ext

	return zapcore.AddSync(cronowriter.MustNew(path))
}

func (l *Logger) ToWriter(level zapcore.Level) io.Writer {
	return &zapio.Writer{
		Log:   l.Logger,
		Level: level,
	}
}

func (l *Logger) ZapLogger() *zap.Logger {
	return l.Logger
}

func (l *Logger) Clone() *Logger {
	return &Logger{
		Logger:        l.Logger,
		FilePath:      l.FilePath,
		ErrorFilePath: l.ErrorFilePath,
		fileLevel:     l.fileLevel,
		consoleLevel:  l.consoleLevel,
		fileEncoder:   l.fileEncoder,
	}
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	_l := l.Clone()
	_l.Logger = l.Logger.With(fields...)
	return _l
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

type fileEncoder struct {
	zapcore.Encoder
}
