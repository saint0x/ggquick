package log

import (
	"fmt"
	"strings"
)

// Color codes
const (
	reset      = "\033[0m"
	bold       = "\033[1m"
	dim        = "\033[2m"
	red        = "\033[31m"
	green      = "\033[32m"
	yellow     = "\033[33m"
	blue       = "\033[34m"
	magenta    = "\033[35m"
	cyan       = "\033[36m"
	white      = "\033[37m"
	boldRed    = "\033[1;31m"
	boldGreen  = "\033[1;32m"
	boldYellow = "\033[1;33m"
	boldBlue   = "\033[1;34m"
)

// Emojis for different log types
const (
	infoEmoji    = "ℹ️ "
	successEmoji = "✅ "
	errorEmoji   = "❌ "
	warnEmoji    = "⚠️ "
	stepEmoji    = "👉 "
	debugEmoji   = "🔍 "
	prEmoji      = "🔄 "
	gitEmoji     = "📦 "
	branchEmoji  = "🌿 "
	diffEmoji    = "📝 "
)

// Logger struct with debug flag
type Logger struct {
	debug bool
}

// New creates a new logger instance
func New(debug bool) *Logger {
	return &Logger{debug: debug}
}

// formatMessage adds padding and wraps long lines
func formatMessage(msg string) string {
	width := 80
	lines := strings.Split(msg, "\n")
	var formatted []string

	for _, line := range lines {
		if len(line) <= width {
			formatted = append(formatted, line)
			continue
		}

		words := strings.Fields(line)
		current := ""
		for _, word := range words {
			if len(current)+len(word)+1 > width {
				formatted = append(formatted, current)
				current = word
			} else {
				if current == "" {
					current = word
				} else {
					current += " " + word
				}
			}
		}
		if current != "" {
			formatted = append(formatted, current)
		}
	}

	return strings.Join(formatted, "\n")
}

// Info prints an info message
func (l *Logger) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s%s\n", blue, infoEmoji, formatMessage(msg), reset)
}

// Success prints a success message
func (l *Logger) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s%s\n", boldGreen, successEmoji, formatMessage(msg), reset)
}

// Error prints an error message
func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s%s\n", boldRed, errorEmoji, formatMessage(msg), reset)
}

// Warning prints a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s%s\n", boldYellow, warnEmoji, formatMessage(msg), reset)
}

// Step prints a step message
func (l *Logger) Step(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s%s\n", cyan, stepEmoji, formatMessage(msg), reset)
}

// Debug prints a debug message if debug is enabled
func (l *Logger) Debug(format string, args ...interface{}) {
	if !l.debug {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s%s\n", dim, debugEmoji, formatMessage(msg), reset)
}

// PR prints a PR-related message
func (l *Logger) PR(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s%s\n", magenta, prEmoji, formatMessage(msg), reset)
}

// Git prints a git-related message
func (l *Logger) Git(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s%s\n", white, gitEmoji, formatMessage(msg), reset)
}

// Branch prints a branch-related message
func (l *Logger) Branch(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s%s\n", green, branchEmoji, formatMessage(msg), reset)
}

// Diff prints a diff-related message
func (l *Logger) Diff(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s%s\n", yellow, diffEmoji, formatMessage(msg), reset)
}

// IsDebug returns whether debug logging is enabled
func (l *Logger) IsDebug() bool {
	return l.debug
}
