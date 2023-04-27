package logger

import (
	"github.com/utahta/go-cronowriter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/go-mixed/go-common.v1/utils/core"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type Logger struct {
	logger  *zap.Logger
	fields  []zapcore.Field
	options []zap.Option

	FilePath      string
	ErrorFilePath string

	consoleLevel   zapcore.Level
	consoleEncoder zapcore.Encoder
	fileLevel      zapcore.Level
	fileEncoder    zapcore.Encoder
	fileWriters    *sync.Map // filename: io.Writer 如果filename在本列表中存在，需要先关闭，再重新打开

	buildOnce *sync.Once
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

		fileWriters: &sync.Map{},
		buildOnce:   &sync.Once{},
	}

	logger.options = []zap.Option{
		zap.WithCaller(true),
		zap.AddCallerSkip(1), // 当前类的Debug、Info是封装函数，frame多了一层，故此处+1
		zap.AddStacktrace(zap.LevelEnablerFunc(errorLevelFn)),
	}

	logger.getLogger()

	return logger, nil
}

// SetFileLevel 修改文件输出的最低Level，可实时修改。无法关闭ErrorLevel及以上的输出
//
//	如果不想输出Info、Debug、Warn，可以这样: SetFileLevel(zap.ErrorLevel)
func (log *Logger) SetFileLevel(level zapcore.Level) *Logger {
	log.fileLevel = level
	return log
}

// SetConsoleLevel 修改控制台输出的最低Level，可实时修改。无法关闭ErrorLevel及以上的输出
//
//	如果不想输出Info、Debug、Warn，可以这样: SetConsoleLevel(zap.ErrorLevel)
func (log *Logger) SetConsoleLevel(level zapcore.Level) *Logger {
	log.consoleLevel = level
	return log
}

// SetFileEncoder 修改文件输出的encoder：console、json，可实时修改
func (log *Logger) SetFileEncoder(encoder string) *Logger {
	log.fileEncoder = makeEncoder(encoder)
	return log
}

func (log *Logger) buildZapCores() (cores *zapCores) {
	cores = &zapCores{}
	errorLevelFunc := zap.LevelEnablerFunc(errorLevelFn)

	// 获取console的cores
	consoleEncoderFunc := encoderFunc(log.consoleEncoderFunc)
	consoleLevelFunc := zap.LevelEnablerFunc(log.consoleLevelFunc)
	cores.ConsoleNormalLevelCore = zapcore.NewCore(consoleEncoderFunc, zapcore.AddSync(os.Stdout), consoleLevelFunc) // stdout输出
	cores.ConsoleErrorLevelCore = zapcore.NewCore(consoleEncoderFunc, zapcore.AddSync(os.Stderr), errorLevelFunc)    // stderr输出

	// 获取文件的cores
	fileSyncer := log.getFileWriter(log.FilePath)
	fileLevelFunc := zap.LevelEnablerFunc(log.fileLevelFunc)
	fileEncoderFunc := encoderFunc(log.fileEncoderFunc)

	cores.FileNormalLevelCore = zapcore.NewCore(fileEncoderFunc, fileSyncer, fileLevelFunc) // 常规信息的文件输出

	if log.ErrorFilePath != "" {
		errorSyncer := log.getFileWriter(log.ErrorFilePath)
		cores.FileErrorLevelCore = zapcore.NewCore(fileEncoderFunc, errorSyncer, errorLevelFunc) // 错误log单独输出
	} else {
		cores.FileErrorLevelCore = zapcore.NewCore(fileEncoderFunc, fileSyncer, errorLevelFunc) // 错误和常规的log在一个文件
	}

	return cores
}

func (log *Logger) getLogger() *zap.Logger {
	if log.logger != nil {
		return log.logger
	}

	if log.buildOnce != nil {
		log.buildOnce.Do(func() {
			log.logger = zap.New(
				log.buildZapCores().With(log.fields), // 附加的fields
				log.options...,                       // 附加的options
			)
		})

		log.buildOnce = nil // GC
	}

	return log.logger
}

func (log *Logger) consoleEncoderFunc() zapcore.Encoder {
	return log.consoleEncoder
}

func (log *Logger) fileEncoderFunc() zapcore.Encoder {
	return log.fileEncoder
}

func (log *Logger) fileLevelFunc(level zapcore.Level) bool {
	return level >= log.fileLevel && level < zap.ErrorLevel
}

func (log *Logger) consoleLevelFunc(level zapcore.Level) bool {
	return level >= log.consoleLevel && level < zap.ErrorLevel
}

func (log *Logger) buildIOWriter(filename string) io.WriteCloser {
	ext := filepath.Ext(filename)
	path := filename[0:len(filename)-len(ext)] + ".%Y-%m-%d" + ext

	return cronowriter.MustNew(path)
}

func (log *Logger) getFileWriter(filename string) zapcore.WriteSyncer {
	_writer, ok := log.fileWriters.Load(filename)
	// 已存在writer，直接返回
	if ok {
		return zapcore.AddSync(_writer.(io.Writer))
	}

	// 不存在writer，创建并保存
	writer := log.buildIOWriter(filename)
	log.fileWriters.Store(filename, writer)

	return zapcore.AddSync(writer)
}

func (log *Logger) ZapLogger() *zap.Logger {
	// 返回给外部使用的zap logger, 需要减少一层的frame
	return log.AddCallerSkip(-1).getLogger()
}

func (log *Logger) AddCallerSkip(skip int) *Logger {
	return log.WithOptions(zap.AddCallerSkip(skip))
}

func (log *Logger) clone() *Logger {
	return &Logger{
		fields:  core.CopyFrom(log.fields),
		options: core.CopyFrom(log.options),

		FilePath:      log.FilePath,
		ErrorFilePath: log.ErrorFilePath,

		consoleLevel:   log.consoleLevel,
		consoleEncoder: log.consoleEncoder,
		fileLevel:      log.fileLevel,
		fileEncoder:    log.fileEncoder,

		fileWriters: log.fileWriters,
		buildOnce:   &sync.Once{},
	}
}
