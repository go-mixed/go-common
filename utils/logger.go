package utils

import (
	"github.com/utahta/go-cronowriter"
	"go-common/utils/core"
	io_utils "go-common/utils/io"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapio"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DEBUG = iota
	INFO
	WARN
	PANIC
	ERROR
	FATAL
)

type LoggerOptions struct {
	FilePath      string `json:"file_path" yaml:"file_path" validate:"required"`
	ErrorFilePath string `json:"error_file_path" yaml:"error_file_path"`

	FileEncoder     string `json:"file_encoder" yaml:"file_encoder"`
	FileMinLevel    string `json:"file_min_level" yaml:"file_min_level"`
	ConsoleMinLevel string `json:"console_min_level" yaml:"console_min_level"`
}

func DefaultLoggerOptions() LoggerOptions {
	return LoggerOptions{
		FilePath:      filepath.Join(io_utils.GetCurrentDir(), "logs", "app.log"),
		ErrorFilePath: "",

		FileEncoder:     "console",
		FileMinLevel:    "debug",
		ConsoleMinLevel: "debug",
	}
}

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

	consoleLevel   zapcore.Level
	consoleEncoder zapcore.Encoder
	fileLevel      zapcore.Level
	fileEncoder    zapcore.Encoder
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

// NewLogger 新建一个独立的logger
func NewLogger(options LoggerOptions) (*Logger, error) {
	// 创建文件夹
	if err := os.MkdirAll(filepath.Dir(options.FilePath), os.ModePerm); err != nil {
		return nil, err
	}
	if options.ErrorFilePath != "" {
		if err := os.MkdirAll(filepath.Dir(options.ErrorFilePath), os.ModePerm); err != nil {
			return nil, err
		}
	}

	logger := &Logger{
		FilePath:      options.FilePath,
		ErrorFilePath: options.ErrorFilePath,

		consoleLevel:   toZapLevel(core.If(options.ConsoleMinLevel != "", options.ConsoleMinLevel, os.Getenv(ZapConsoleLevel))),
		consoleEncoder: makeEncoder("console"),
		fileLevel:      toZapLevel(core.If(options.FileMinLevel != "", options.FileMinLevel, os.Getenv(ZapFileLevel))),
		fileEncoder:    makeEncoder(core.If(options.FileEncoder != "", options.FileEncoder, os.Getenv(ZapFileEncoder))),
	}

	logger.Logger = zap.New(
		logger.buildCore(),
		zap.AddCaller(),
		zap.AddStacktrace(zap.LevelEnablerFunc(logger.errorLevelFunc)),
	)

	return logger, nil
}

// InitLogger 全局新建Logger
// errorFilename 非空时表示将ERROR以上的日志写入到此文件中
func InitLogger(filename string, errorFilename string) (*Logger, error) {
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

// SetFileLevel 修改文件输出的最低Level，可实时修改。无法关闭ErrorLevel及以上的输出
//
//	如果不想输出Info、Debug、Warn，可以这样: SetFileLevel(zap.ErrorLevel)
func (l *Logger) SetFileLevel(level zapcore.Level) *Logger {
	l.fileLevel = level
	return l
}

// SetConsoleLevel 修改控制台输出的最低Level，可实时修改。无法关闭ErrorLevel及以上的输出
//
//	如果不想输出Info、Debug、Warn，可以这样: SetConsoleLevel(zap.ErrorLevel)
func (l *Logger) SetConsoleLevel(level zapcore.Level) *Logger {
	l.consoleLevel = level
	return l
}

// SetFileEncoder 修改文件输出的encoder：console、json，可实时修改
func (l *Logger) SetFileEncoder(encoder string) *Logger {
	l.fileEncoder = makeEncoder(encoder)
	return l
}

func (l *Logger) buildCore() zapcore.Core {

	var cores []zapcore.Core
	errorLevelFunc := zap.LevelEnablerFunc(l.errorLevelFunc)

	// 获取console的cores
	consoleEncoderFunc := encoderFunc(l.consoleEncoderFunc)
	consoleLevelFunc := zap.LevelEnablerFunc(l.consoleLevelFunc)
	stdoutSyncer := zapcore.AddSync(os.Stdout)
	stderrSyncer := zapcore.AddSync(os.Stderr)
	cores = append(cores,
		zapcore.NewCore(consoleEncoderFunc, stdoutSyncer, consoleLevelFunc), // stdout输出
		zapcore.NewCore(consoleEncoderFunc, stderrSyncer, errorLevelFunc),   // stderr输出
	)

	// 获取文件的cores
	fileSyncer := l.getFileWriter(l.FilePath)
	fileLevelFunc := zap.LevelEnablerFunc(l.fileLevelFunc)
	fileEncoderFunc := encoderFunc(l.fileEncoderFunc)
	cores = append(cores, zapcore.NewCore(fileEncoderFunc, fileSyncer, fileLevelFunc)) // 常规信息的文件输出

	if l.ErrorFilePath != "" {
		errorSyncer := l.getFileWriter(l.ErrorFilePath)
		cores = append(cores, zapcore.NewCore(fileEncoderFunc, errorSyncer, errorLevelFunc)) // 错误log单独输出
	} else {
		cores = append(cores, zapcore.NewCore(fileEncoderFunc, fileSyncer, errorLevelFunc)) // 错误和常规的log在一个文件
	}

	return zapcore.NewTee(cores...)
}

func (l *Logger) consoleEncoderFunc() zapcore.Encoder {
	return l.consoleEncoder
}

func (l *Logger) fileEncoderFunc() zapcore.Encoder {
	return l.fileEncoder
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

		consoleLevel:   l.consoleLevel,
		consoleEncoder: l.consoleEncoder,
		fileLevel:      l.fileLevel,
		fileEncoder:    l.fileEncoder,
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

// 默认是INFO
func toZapLevel(level string) zapcore.Level {
	minLevel := zapcore.DebugLevel
	envLevel := strings.ToLower(level)
	if envLevel == "disabled" {
		return zap.ErrorLevel // 设置为ERROR会跳过DEBUG、INFO、WARN
	} else {
		if err := minLevel.UnmarshalText([]byte(envLevel)); err != nil {
			minLevel = zap.DebugLevel
		}
	}

	return minLevel
}

func makeEncoder(encoder string) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	switch encoder {
	case "json", "JSON":
		return zapcore.NewJSONEncoder(encoderConfig)
	default:
		return zapcore.NewConsoleEncoder(encoderConfig)
	}
}

type encoderFunc func() zapcore.Encoder

func (f encoderFunc) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	return f().AddArray(key, marshaler)
}

func (f encoderFunc) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	return f().AddObject(key, marshaler)
}

func (f encoderFunc) AddBinary(key string, value []byte) {
	f().AddBinary(key, value)
}

func (f encoderFunc) AddByteString(key string, value []byte) {
	f().AddByteString(key, value)
}

func (f encoderFunc) AddBool(key string, value bool) {
	f().AddBool(key, value)
}

func (f encoderFunc) AddComplex128(key string, value complex128) {
	f().AddComplex128(key, value)
}

func (f encoderFunc) AddComplex64(key string, value complex64) {
	f().AddComplex64(key, value)
}

func (f encoderFunc) AddDuration(key string, value time.Duration) {
	f().AddDuration(key, value)
}

func (f encoderFunc) AddFloat64(key string, value float64) {
	f().AddFloat64(key, value)
}

func (f encoderFunc) AddFloat32(key string, value float32) {
	f().AddFloat32(key, value)
}

func (f encoderFunc) AddInt(key string, value int) {
	f().AddInt(key, value)
}

func (f encoderFunc) AddInt64(key string, value int64) {
	f().AddInt64(key, value)
}

func (f encoderFunc) AddInt32(key string, value int32) {
	f().AddInt32(key, value)
}

func (f encoderFunc) AddInt16(key string, value int16) {
	f().AddInt16(key, value)
}

func (f encoderFunc) AddInt8(key string, value int8) {
	f().AddInt8(key, value)
}

func (f encoderFunc) AddString(key, value string) {
	f().AddString(key, value)
}

func (f encoderFunc) AddTime(key string, value time.Time) {
	f().AddTime(key, value)
}

func (f encoderFunc) AddUint(key string, value uint) {
	f().AddUint(key, value)
}

func (f encoderFunc) AddUint64(key string, value uint64) {
	f().AddUint64(key, value)
}

func (f encoderFunc) AddUint32(key string, value uint32) {
	f().AddUint32(key, value)
}

func (f encoderFunc) AddUint16(key string, value uint16) {
	f().AddUint16(key, value)
}

func (f encoderFunc) AddUint8(key string, value uint8) {
	f().AddUint8(key, value)
}

func (f encoderFunc) AddUintptr(key string, value uintptr) {
	f().AddUintptr(key, value)
}

func (f encoderFunc) AddReflected(key string, value interface{}) error {
	return f().AddReflected(key, value)
}

func (f encoderFunc) OpenNamespace(key string) {
	f().OpenNamespace(key)
}

func (f encoderFunc) Clone() zapcore.Encoder {
	return f().Clone()
}

func (f encoderFunc) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	return f().EncodeEntry(entry, fields)
}
