package util

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Colors
const (
	ColorReset     = "\033[0m"
	ColorRed       = "\033[31m"
	ColorGreen     = "\033[32m"
	ColorYellow    = "\033[33m"
	ColorBlue      = "\033[36m"
	ColorGray      = "\033[90m"
	ColorBold      = "\033[1m"
	ColorLightBlue = "\033[94m"
)

// Emojis
const (
	EmojiRocket  = "🚀"
	EmojiStar    = "⭐"
	EmojiError   = "❌"
	EmojiCheck   = "✅"
	EmojiKey     = "🔑"
	EmojiGear    = "⚙️"
	EmojiLink    = "🔗"
	EmojiWave    = "👋"
	EmojiWarning = "⚠️"
	EmojiAPI     = "🔍"
	EmojiDone    = "🎯"
	EmojiSelect  = "🔘"

	// emoji for loading
	EmojiLoading = "🌐"

	EmojiCongratulation = "🎉"
	EmojiDiamond        = "💎"
)

// Common formatting
const (
	SeparatorChar  = "-"
	SeparatorWidth = 80
)

// GetSeparator returns a separator line of standard width
func GetSeparator() string {
	return strings.Repeat(SeparatorChar, SeparatorWidth)
}

func ClearConsole() {
	fmt.Print("\033[H\033[2J")
}

// Printer handles output formatting with configurable writer
type Printer struct {
	out io.Writer
}

// NewPrinter creates a new Printer with the given writer
func NewPrinter(w io.Writer) *Printer {
	if w == nil {
		w = os.Stdout
	}
	return &Printer{out: w}
}

// PrintTitle prints a title with an emoji and separator
func (p *Printer) PrintTitle(title string, emoji string) {
	fmt.Fprintf(p.out, "\n%s %s%s%s", emoji, ColorBold, title, ColorReset)
	p.PrintSeparator()
}

const maxErrorLength = 300

// PrintError prints an error message
func (p *Printer) PrintError(message string) {
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > maxErrorLength {
		message = message[:maxErrorLength-3] + "..."
	}
	fmt.Fprintf(p.out, "%s%s %s%s\n", ColorRed, EmojiError, message, ColorReset)
}

// PrintSuccess prints a success message
func (p *Printer) PrintSuccess(message string) {
	fmt.Fprintf(p.out, "\n%s%s %s%s\n", ColorGreen, EmojiDone, message, ColorReset)
}

// PrintWarning prints a warning message
func (p *Printer) PrintWarning(message string) {
	fmt.Fprintf(p.out, "%s%s %s%s\n", ColorYellow, EmojiWarning, message, ColorReset)
}

// FormatTitle formats a title with an emoji
func (p *Printer) FormatTitle(title string, emoji string) string {
	return fmt.Sprintf("\n%s %s%s%s\n", emoji, ColorBold, title, ColorReset)
}

// Printf formats and prints a message
func (p *Printer) Printf(format string, args ...interface{}) {
	fmt.Fprintf(p.out, format, args...)
}

// Println prints a message with a newline
func (p *Printer) Println(args ...interface{}) {
	fmt.Fprintln(p.out, args...)
}

// Print prints a message
func (p *Printer) Print(args ...interface{}) {
	fmt.Fprint(p.out, args...)
}

// PrintSeparator prints a separator line
func (p *Printer) PrintSeparator() {
	p.Printf("\n%s\n", GetSeparator())
}

func (p *Printer) PrintTesting() {
	msg := "测试中,请稍等..."
	fmt.Printf("\n%s %s\n\n", EmojiLoading, msg)
}
