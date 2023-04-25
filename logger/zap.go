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

func (log *Logger) With(fields ...zap.Field) *Logger {
	_l := log.clone()
	_l.fields = append(_l.fields, fields...)
	return _l
}

func (log *Logger) WithOptions(options ...zap.Option) *Logger {
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
func (log *Logger) Log(lvl zapcore.Level, msg string, fields ...zap.Field) {
	log.getLogger().Log(lvl, msg, fields...)
}

// Debug logs a message at DebugLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Debug(msg string, fields ...zap.Field) {
	log.getLogger().Debug(msg, fields...)
}

// Info logs a message at InfoLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Info(msg string, fields ...zap.Field) {
	log.getLogger().Info(msg, fields...)
}

// Warn logs a message at WarnLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Warn(msg string, fields ...zap.Field) {
	log.getLogger().Warn(msg, fields...)
}

// Error logs a message at ErrorLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Error(msg string, fields ...zap.Field) {
	log.getLogger().Error(msg, fields...)
}

// DPanic logs a message at DPanicLevel. The message includes any fields
// passed at the log site, as well as any fields accumulated on the logger.
//
// If the logger is in development mode, it then panics (DPanic means
// "development panic"). This is useful for catching errors that are
// recoverable, but shouldn't ever happen.
func (log *Logger) DPanic(msg string, fields ...zap.Field) {
	log.getLogger().DPanic(msg, fields...)
}

// Panic logs a message at PanicLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then panics, even if logging at PanicLevel is disabled.
func (log *Logger) Panic(msg string, fields ...zap.Field) {
	log.getLogger().Panic(msg, fields...)
}

// Fatal logs a message at FatalLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then calls os.Exit(1), even if logging at FatalLevel is
// disabled.
func (log *Logger) Fatal(msg string, fields ...zap.Field) {
	log.getLogger().Fatal(msg, fields...)
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
