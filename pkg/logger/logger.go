package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"payflow/pkg/tracing"
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

	WithContext(ctx context.Context) Logger
	DebugContext(ctx context.Context, msg string, fields map[string]interface{})
	InfoContext(ctx context.Context, msg string, fields map[string]interface{})
	WarnContext(ctx context.Context, msg string, fields map[string]interface{})
	ErrorContext(ctx context.Context, msg string, fields map[string]interface{})

	WithFields(fields map[string]interface{}) Logger
}

type ZerologLogger struct {
	logger zerolog.Logger
	fields map[string]interface{}
}

func New(level LogLevel, output io.Writer) Logger {
	if output == nil {
		output = os.Stdout
	}

	zerolog.TimeFieldFormat = time.RFC3339
	zerologLevel := getZerologLevel(level)

	var consoleWriter io.Writer
	if strings.ToLower(os.Getenv("APP_ENV")) == "development" {
		consoleWriter = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	} else {
		consoleWriter = output
	}

	zl := zerolog.New(consoleWriter).
		Level(zerologLevel).
		With().
		Timestamp().
		Logger()

	return &ZerologLogger{
		logger: zl,
		fields: make(map[string]interface{}),
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

func (l *ZerologLogger) WithFields(fields map[string]interface{}) Logger {
	newLogger := &ZerologLogger{
		logger: l.logger,
		fields: make(map[string]interface{}, len(l.fields)+len(fields)),
	}

	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

func (l *ZerologLogger) WithContext(ctx context.Context) Logger {
	newLogger := &ZerologLogger{
		logger: l.logger,
		fields: make(map[string]interface{}, len(l.fields)+2), // Trace ve Span ID için
	}

	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	traceID := tracing.GetTraceID(ctx)
	if traceID != "" {
		newLogger.fields["trace_id"] = traceID
	}

	return newLogger
}

func (l *ZerologLogger) addSourceInfo(event *zerolog.Event) *zerolog.Event {
	if l.logger.GetLevel() == zerolog.DebugLevel {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			parts := strings.Split(file, "/")
			if len(parts) > 2 {
				file = strings.Join(parts[len(parts)-2:], "/")
			}
			event = event.Str("source", fmt.Sprintf("%s:%d", file, line))
		}
	}
	return event
}

func (l *ZerologLogger) Debug(msg string, fields map[string]interface{}) {
	event := l.addSourceInfo(l.logger.Debug())
	for k, v := range l.fields {
		event = event.Interface(k, v)
	}
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) Info(msg string, fields map[string]interface{}) {
	event := l.logger.Info()
	for k, v := range l.fields {
		event = event.Interface(k, v)
	}
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) Warn(msg string, fields map[string]interface{}) {
	event := l.logger.Warn()
	for k, v := range l.fields {
		event = event.Interface(k, v)
	}
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) Error(msg string, fields map[string]interface{}) {
	event := l.logger.Error()
	for k, v := range l.fields {
		event = event.Interface(k, v)
	}
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) Fatal(msg string, fields map[string]interface{}) {
	event := l.logger.Fatal()
	for k, v := range l.fields {
		event = event.Interface(k, v)
	}
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) Panic(msg string, fields map[string]interface{}) {
	event := l.logger.Panic()
	for k, v := range l.fields {
		event = event.Interface(k, v)
	}
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *ZerologLogger) DebugContext(ctx context.Context, msg string, fields map[string]interface{}) {
	l.WithContext(ctx).Debug(msg, fields)
}

func (l *ZerologLogger) InfoContext(ctx context.Context, msg string, fields map[string]interface{}) {
	l.WithContext(ctx).Info(msg, fields)
}

func (l *ZerologLogger) WarnContext(ctx context.Context, msg string, fields map[string]interface{}) {
	l.WithContext(ctx).Warn(msg, fields)
}

func (l *ZerologLogger) ErrorContext(ctx context.Context, msg string, fields map[string]interface{}) {
	l.WithContext(ctx).Error(msg, fields)
}

func FormatError(err error) string {
	if err != nil {
		return fmt.Sprintf("Hata: %v", err)
	}
	return ""
}
