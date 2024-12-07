package log

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Default handler outputting to stderr.
var Default = NewText(os.Stderr)

// start time.
var start = time.Now()

// colors.
const (
	none   = 0
	red    = 31
	green  = 32
	yellow = 33
	blue   = 34
	gray   = 37
)

// Colors mapping.
var Colors = [...]int{
	DebugLevel: gray,
	InfoLevel:  blue,
	WarnLevel:  yellow,
	ErrorLevel: red,
	FatalLevel: red,
}

// Strings mapping.
var Strings = [...]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
}

// Handler implementation.
type TextHandler struct {
	mu     sync.Mutex
	Writer io.Writer
}

// NewText handler.
func NewText(w io.Writer) Handler {
	return &TextHandler{
		Writer: w,
	}
}

// HandleLog implements Handler.
func (h *TextHandler) HandleLog(e *Entry) error {
	color := Colors[e.Level]
	level := Strings[e.Level]
	names := e.Fields.Names()

	h.mu.Lock()
	defer h.mu.Unlock()

	ts := time.Since(start) / time.Second
	fmt.Fprintf(h.Writer, "\033[%dm%6s\033[0m[%04d] %-25s", color, level, ts, e.Message)

	for _, name := range names {
		fmt.Fprintf(h.Writer, " \033[%dm%s\033[0m=%v", color, name, e.Fields.Get(name))
	}

	fmt.Fprintln(h.Writer)

	return nil
}
