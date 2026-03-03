package logger

import (
	"log"
	"os"
)

// Logger is a simple leveled logger backed by the standard log package.
// It supports INFO, DEBUG, and ERROR levels with printf-style formatting.
// DEBUG messages are suppressed unless the logger was created with debug=true.
type Logger struct {
	debug  bool
	prefix string
	l      *log.Logger
}

// New creates a Logger that writes to stdout.
// If debug is true, Debugf messages are emitted.
func New(debug bool) *Logger {
	return &Logger{
		debug: debug,
		l:     log.New(os.Stdout, "", log.LstdFlags),
	}
}

// With returns a child Logger that prepends prefix to every log line.
func (l *Logger) With(prefix string) *Logger {
	return &Logger{
		debug:  l.debug,
		prefix: l.prefix + prefix + " ",
		l:      l.l,
	}
}

// Infof logs a formatted message at INFO level.
func (l *Logger) Infof(format string, args ...any) {
	l.l.Printf("INFO  "+l.prefix+format, args...)
}

// Debugf logs a formatted message at DEBUG level.
// The message is discarded when the logger was created with debug=false.
func (l *Logger) Debugf(format string, args ...any) {
	if !l.debug {
		return
	}
	l.l.Printf("DEBUG "+l.prefix+format, args...)
}

// Errorf logs a formatted message at ERROR level.
func (l *Logger) Errorf(format string, args ...any) {
	l.l.Printf("ERROR "+l.prefix+format, args...)
}

// Write implements io.Writer at DEBUG level so the logger can be plugged into
// places that expect an io.Writer (e.g. go-smtp raw protocol tracing).
func (l *Logger) Write(p []byte) (n int, err error) {
	if !l.debug {
		return len(p), nil
	}
	line := string(p)
	// strip trailing newlines — log.Logger adds its own
	for line != "" && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
		line = line[:len(line)-1]
	}
	if line != "" {
		l.l.Printf("DEBUG "+l.prefix+"%s", line)
	}
	return len(p), nil
}
