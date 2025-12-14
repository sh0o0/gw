package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	warnColor    = color.New(color.FgYellow)
	infoColor    = color.New(color.FgCyan)
	dimColor     = color.New(color.Faint)
	boldColor    = color.New(color.Bold)
)

const (
	emojiSuccess = "✅"
	emojiError   = "❌"
	emojiWarn    = "⚠️ "
	emojiInfo    = "ℹ️ "
	emojiWorking = "🔧"
	emojiBranch  = "🌿"
	emojiFolder  = "📁"
	emojiLink    = "🔗"
	emojiTrash   = "🗑️ "
	emojiRocket  = "🚀"
)

type Output struct {
	w io.Writer
}

func NewOutput() *Output {
	return &Output{w: os.Stderr}
}

func (o *Output) Success(format string, args ...any) {
	successColor.Fprintf(o.w, "%s %s\n", emojiSuccess, fmt.Sprintf(format, args...))
}

func (o *Output) Error(format string, args ...any) {
	errorColor.Fprintf(o.w, "%s %s\n", emojiError, fmt.Sprintf(format, args...))
}

func (o *Output) Warn(format string, args ...any) {
	warnColor.Fprintf(o.w, "%s %s\n", emojiWarn, fmt.Sprintf(format, args...))
}

func (o *Output) Info(format string, args ...any) {
	infoColor.Fprintf(o.w, "%s %s\n", emojiInfo, fmt.Sprintf(format, args...))
}

func (o *Output) Working(format string, args ...any) {
	infoColor.Fprintf(o.w, "%s %s\n", emojiWorking, fmt.Sprintf(format, args...))
}

func (o *Output) Branch(format string, args ...any) {
	fmt.Fprintf(o.w, "%s %s\n", emojiBranch, fmt.Sprintf(format, args...))
}

func (o *Output) Folder(format string, args ...any) {
	fmt.Fprintf(o.w, "%s %s\n", emojiFolder, fmt.Sprintf(format, args...))
}

func (o *Output) Link(format string, args ...any) {
	dimColor.Fprintf(o.w, "%s %s\n", emojiLink, fmt.Sprintf(format, args...))
}

func (o *Output) Trash(format string, args ...any) {
	fmt.Fprintf(o.w, "%s %s\n", emojiTrash, fmt.Sprintf(format, args...))
}

func (o *Output) Summary(success, failed int, noun string) {
	fmt.Fprintln(o.w)
	boldColor.Fprintln(o.w, "━━━ Summary ━━━")
	if success > 0 {
		successColor.Fprintf(o.w, "  %s %d %s(s) succeeded\n", emojiSuccess, success, noun)
	}
	if failed > 0 {
		errorColor.Fprintf(o.w, "  %s %d %s(s) failed\n", emojiError, failed, noun)
	}
}

func (o *Output) Highlight(s string) string {
	return boldColor.Sprint(s)
}

func (o *Output) Dim(s string) string {
	return dimColor.Sprint(s)
}

var out = NewOutput()
