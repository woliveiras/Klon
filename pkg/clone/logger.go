package clone

import "log"

// Logger is a minimal logging interface used throughout the clone package.
// It matches the stdlib log Logger for Printf/Println.
type Logger interface {
	Printf(string, ...any)
	Println(...any)
}

var logSink Logger = log.Default()

// SetLogger allows callers/tests to inject a custom logger (or noop) instead of
// the default stdlib logger. Passing nil resets to the default.
func SetLogger(l Logger) {
	if l == nil {
		logSink = log.Default()
		return
	}
	logSink = l
}
