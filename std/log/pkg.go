package log

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"time"
)

// field used for sorting.
type field struct {
	Name  string
	Value interface{}
}

// by sorts fields by name.
type byName []field

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }

// handleStdLog outpouts to the stlib log.
func handleStdLog(e *Entry) error {
	level := levelNames[e.Level]

	var fields []field

	for k, v := range e.Fields {
		fields = append(fields, field{k, v})
	}

	sort.Sort(byName(fields))

	var b bytes.Buffer
	fmt.Fprintf(&b, "%5s %-25s", level, e.Message)

	for _, f := range fields {
		fmt.Fprintf(&b, " %s=%v", f.Name, f.Value)
	}

	log.Println(b.String())

	return nil
}

// singletons ftw?
var Log Interface = &Logger{
	Handler: HandlerFunc(handleStdLog),
	Level:   InfoLevel,
}

// SetHandler sets the handler. This is not thread-safe.
// The default handler outputs to the stdlib log.
func SetHandler(h Handler) {
	if logger, ok := Log.(*Logger); ok {
		logger.Handler = h
	}
}

// SetLevel sets the log level. This is not thread-safe.
func SetLevel(l Level) {
	if logger, ok := Log.(*Logger); ok {
		logger.Level = l
	}
}

// SetLevelFromString sets the log level from a string, panicing when invalid. This is not thread-safe.
func SetLevelFromString(s string) {
	if logger, ok := Log.(*Logger); ok {
		logger.Level = MustParseLevel(s)
	}
}

// WithFields returns a new entry with `fields` set.
func WithFields(fields Fielder) *Entry {
	return Log.WithFields(fields)
}

// WithField returns a new entry with the `key` and `value` set.
func WithField(key string, value interface{}) *Entry {
	return Log.WithField(key, value)
}

// WithDuration returns a new entry with the "duration" field set
// to the given duration in milliseconds.
func WithDuration(d time.Duration) *Entry {
	return Log.WithDuration(d)
}

// WithError returns a new entry with the "error" set to `err`.
func WithError(err error) *Entry {
	return Log.WithError(err)
}

// Debug level message.
func Debug(msg string) {
	Log.Debug(msg)
}

// Info level message.
func Info(msg string) {
	Log.Info(msg)
}

// Warn level message.
func Warn(msg string) {
	Log.Warn(msg)
}

// Error level message.
func Error(msg string) {
	Log.Error(msg)
}

// Fatal level message, followed by an exit.
func Fatal(msg string) {
	Log.Fatal(msg)
}

// Debugf level formatted message.
func Debugf(msg string, v ...interface{}) {
	Log.Debugf(msg, v...)
}

// Infof level formatted message.
func Infof(msg string, v ...interface{}) {
	Log.Infof(msg, v...)
}

// Warnf level formatted message.
func Warnf(msg string, v ...interface{}) {
	Log.Warnf(msg, v...)
}

// Errorf level formatted message.
func Errorf(msg string, v ...interface{}) {
	Log.Errorf(msg, v...)
}

// Fatalf level formatted message, followed by an exit.
func Fatalf(msg string, v ...interface{}) {
	Log.Fatalf(msg, v...)
}

// Trace returns a new entry with a Stop method to fire off
// a corresponding completion log, useful with defer.
func Trace(msg string) *Entry {
	return Log.Trace(msg)
}
