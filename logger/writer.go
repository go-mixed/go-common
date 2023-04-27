package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapio"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"io"
)

func (log *Logger) ToWriter(level zapcore.Level) io.Writer {
	return &zapio.Writer{
		Log:   log.ZapLogger(),
		Level: level,
	}
}

// ILogger wraps the Logger to provide a more ergonomic, but slightly slower,
// API. Sugaring a Logger is quite inexpensive, so it's reasonable for a
// single application to use both Loggers and SugaredLoggers, converting
// between them on the boundaries of performance-sensitive code.
func (log *Logger) ILogger() utils.ILogger {
	return log.ZapLogger().Sugar()
}

// Named adds a new path segment to the logger's name. Segments are joined by
// periods. By default, Loggers are unnamed.
func (log *Logger) Named(s string) *Logger {
	log.getLogger().Named(s)
	return log
}

// 解析fields，支持以下两种格式，可以混合使用，遇到不对称的的情况，会忽略该key
//  1. zap.Field
//  2. [key, value, ...]，key必须是string，value可以是任意类型
//     eg: "key1", "value1", zap.String("key2", "value2"), "key3", "value3", ...
func (log *Logger) parseFields(fields []any) []zap.Field {
	result := make([]zap.Field, 0, len(fields)/2)
	for i := 0; i < len(fields); {
		switch k := fields[i].(type) {
		case zap.Field:
			result = append(result, k)
			i++
		case string:
			if i+1 >= len(fields) { // 最后一个key没有value，忽略
				break
			}
			result = append(result, zap.Any(k, fields[i+1]))
			i += 2
		default: // 不识别的类型，忽略
			i++
		}
	}

	return result
}

func (log *Logger) With(fields ...any) *Logger {
	_l := log.clone()
	_l.fields = append(_l.fields, log.parseFields(fields)...)
	return _l
}

func (log *Logger) WithOptions(options ...Option) *Logger {
	_l := log.clone()
	_l.options = append(_l.options, options...)
	return _l
}

// Level reports the minimum enabled level for this logger.
//
// For NopLoggers, this is [zapcore.InvalidLevel].
func (log *Logger) Level() zapcore.Level {
	return zapcore.LevelOf(log.getLogger().Core())
}

// Check returns a CheckedEntry if logging a message at the specified level
// is enabled. It's a completely optional optimization; in high-performance
// applications, Check can help avoid allocating a slice to hold fields.
func (log *Logger) Check(lvl zapcore.Level, msg string) *zapcore.CheckedEntry {
	return log.getLogger().Check(lvl, msg)
}

// Log logs a message at the specified level. The message includes any fields
// passed at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Log(lvl zapcore.Level, msg string, fields ...any) {
	log.getLogger().Log(lvl, msg, log.parseFields(fields)...)
}

// Debug logs a message at DebugLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Debug(msg string, fields ...any) {
	log.getLogger().Debug(msg, log.parseFields(fields)...)
}

// Info logs a message at InfoLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Info(msg string, fields ...any) {
	log.getLogger().Info(msg, log.parseFields(fields)...)
}

// Warn logs a message at WarnLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Warn(msg string, fields ...any) {
	log.getLogger().Warn(msg, log.parseFields(fields)...)
}

// Error logs a message at ErrorLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Error(msg string, fields ...any) {
	log.getLogger().Error(msg, log.parseFields(fields)...)
}

// DPanic logs a message at DPanicLevel. The message includes any fields
// passed at the log site, as well as any fields accumulated on the logger.
//
// If the logger is in development mode, it then panics (DPanic means
// "development panic"). This is useful for catching errors that are
// recoverable, but shouldn't ever happen.
func (log *Logger) DPanic(msg string, fields ...any) {
	log.getLogger().DPanic(msg, log.parseFields(fields)...)
}

// Panic logs a message at PanicLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then panics, even if logging at PanicLevel is disabled.
func (log *Logger) Panic(msg string, fields ...any) {
	log.getLogger().Panic(msg, log.parseFields(fields)...)
}

// Fatal logs a message at FatalLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then calls os.Exit(1), even if logging at FatalLevel is
// disabled.
func (log *Logger) Fatal(msg string, fields ...any) {
	log.getLogger().Fatal(msg, log.parseFields(fields)...)
}

// Sync calls the underlying Core's Sync method, flushing any buffered log
// entries. Applications should take care to call Sync before exiting.
func (log *Logger) Sync() error {
	return log.getLogger().Sync()
}

// Core returns the Logger's underlying zapcore.Core.
func (log *Logger) Core() zapcore.Core {
	return log.getLogger().Core()
}
