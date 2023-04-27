package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
)

type Option = zap.Option

const ZapConsoleLevel = "ZAP_CONSOLE_LOG_LEVEL"
const ZapFileLevel = "ZAP_LOG_LEVEL"
const ZapFileEncoder = "ZAP_LOG_ENCODER"

func errorLevelFn(level zapcore.Level) bool {
	return level >= zap.ErrorLevel
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
	encoderConfig.EncodeCaller = zapcore.FullCallerEncoder

	switch encoder {
	case "json", "JSON":
		return zapcore.NewJSONEncoder(encoderConfig)
	default:
		return zapcore.NewConsoleEncoder(encoderConfig)
	}
}
