package logger

import "log"

type Logger struct {
	prefix string
}

func New(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

func (l *Logger) Info(format string, args ...any) {
	log.Printf("[INFO] [%s] "+format, append([]any{l.prefix}, args...)...)
}

func (l *Logger) Error(format string, args ...any) {
	log.Printf("[ERROR] [%s] "+format, append([]any{l.prefix}, args...)...)
}
