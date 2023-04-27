package logger

import (
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapCores struct {
	ConsoleNormalLevelCore zapcore.Core
	ConsoleErrorLevelCore  zapcore.Core
	FileNormalLevelCore    zapcore.Core
	FileErrorLevelCore     zapcore.Core
}

var _ zapcore.Core = (*zapCores)(nil)

func (c *zapCores) toSlice() []zapcore.Core {
	return []zapcore.Core{
		c.ConsoleNormalLevelCore,
		c.ConsoleErrorLevelCore,
		c.FileNormalLevelCore,
		c.FileErrorLevelCore,
	}
}

func (c *zapCores) With(fields []zap.Field) zapcore.Core {
	return &zapCores{
		c.ConsoleNormalLevelCore.With(fields),
		c.ConsoleErrorLevelCore.With(fields),
		c.FileNormalLevelCore.With(fields),
		c.FileErrorLevelCore.With(fields),
	}
}

func (c *zapCores) Level() zapcore.Level {
	minLvl := zapcore.FatalLevel // c is never empty

	ls := c.toSlice()
	for i := range ls {
		if lvl := zapcore.LevelOf(ls[i]); lvl < minLvl {
			minLvl = lvl
		}
	}
	return minLvl
}

func (c *zapCores) Enabled(lvl zapcore.Level) bool {
	ls := c.toSlice()

	for i := range ls {
		if ls[i].Enabled(lvl) {
			return true
		}
	}
	return false
}

func (c *zapCores) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	ls := c.toSlice()

	for i := range ls {
		ce = ls[i].Check(ent, ce)
	}
	return ce
}

func (c *zapCores) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	ls := c.toSlice()

	var err error
	for i := range ls {
		err = multierr.Append(err, ls[i].Write(ent, fields))
	}
	return err
}

func (c *zapCores) Sync() error {
	ls := c.toSlice()

	var err error
	for i := range ls {
		err = multierr.Append(err, ls[i].Sync())
	}
	return err
}

func (c *zapCores) Clone() zapcore.Core {
	return c.With(nil)
}

func (c *zapCores) IsNil() bool {
	return c.ConsoleNormalLevelCore == nil || c.ConsoleErrorLevelCore == nil || c.FileNormalLevelCore == nil || c.FileErrorLevelCore == nil
}
