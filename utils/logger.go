package utils

import (
	"log"
	"os"
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
