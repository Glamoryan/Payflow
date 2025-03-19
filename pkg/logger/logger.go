package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
	PanicLevel LogLevel = "panic"
)

type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
	Fatal(msg string, fields map[string]interface{})
	Panic(msg string, fields map[string]interface{})
}

type ZerologLogger struct {
	logger zerolog.Logger
}

func New(level LogLevel, output io.Writer) Logger {
	if output == nil {
		output = os.Stdout
	}

	zerolog.TimeFieldFormat = time.RFC3339
	zerologLevel := getZerologLevel(level)

	zl := zerolog.New(output).
		Level(zerologLevel).
		With().
		Timestamp().
		Logger()

	return &ZerologLogger{
		logger: zl,
	}
}

func getZerologLevel(level LogLevel) zerolog.Level {
	switch strings.ToLower(string(level)) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

func (l *ZerologLogger) Debug(msg string, fields map[string]interface{}) {
	event := l.logger.Debug()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) Info(msg string, fields map[string]interface{}) {
	event := l.logger.Info()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) Warn(msg string, fields map[string]interface{}) {
	event := l.logger.Warn()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) Error(msg string, fields map[string]interface{}) {
	event := l.logger.Error()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) Fatal(msg string, fields map[string]interface{}) {
	event := l.logger.Fatal()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) Panic(msg string, fields map[string]interface{}) {
	event := l.logger.Panic()
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func FormatError(err error) string {
	if err != nil {
		return fmt.Sprintf("Hata: %v", err)
	}
	return ""
}
