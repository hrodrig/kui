package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

type Level int

const (
	Trace Level = iota
	Debug
	Info
	Warn
	Error
	Fatal
	Off
)

var levelNames = map[Level]string{
	Trace: "TRACE",
	Debug: "DEBUG",
	Info:  "INFO",
	Warn:  "WARN",
	Error: "ERROR",
	Fatal: "FATAL",
}

func ParseLevel(s string) (Level, error) {
	switch s {
	case "trace":
		return Trace, nil
	case "debug":
		return Debug, nil
	case "info":
		return Info, nil
	case "warn", "warning":
		return Warn, nil
	case "error":
		return Error, nil
	case "fatal", "critical":
		return Fatal, nil
	case "off", "none":
		return Off, nil
	default:
		return Info, fmt.Errorf("unknown log level: %q", s)
	}
}

type Logger struct {
	level  Level
	logger *log.Logger
}

func New(w io.Writer, level Level) *Logger {
	if w == nil {
		w = os.Stderr
	}
	return &Logger{level: level, logger: log.New(w, "", 0)}
}

func (l *Logger) SetLevel(level Level) { l.level = level }

func (l *Logger) log(level Level, format string, args ...any) {
	if level < l.level {
		return
	}
	msg := fmt.Sprintf(format, args...)
	ts := time.Now().UTC().Format(time.RFC3339)
	l.logger.Printf("%s  %-5s  %s", ts, levelNames[level], msg)
	if level == Fatal {
		os.Exit(1)
	}
}

func (l *Logger) Trace(format string, args ...any) { l.log(Trace, format, args...) }
func (l *Logger) Debug(format string, args ...any) { l.log(Debug, format, args...) }
func (l *Logger) Info(format string, args ...any)  { l.log(Info, format, args...) }
func (l *Logger) Warn(format string, args ...any)  { l.log(Warn, format, args...) }
func (l *Logger) Error(format string, args ...any) { l.log(Error, format, args...) }
func (l *Logger) Fatal(format string, args ...any) { l.log(Fatal, format, args...) }
