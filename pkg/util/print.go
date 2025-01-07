package util

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-coders/check-gpt/pkg/logger"
)

// Colors
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
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

// fmt.Printf("\n%s 正在测试API Key连接性...\n\n", util.EmojiGear)

func (p *Printer) PrintTesting() {
	msg := "测试中,请稍等..."
	fmt.Printf("\n%s %s\n\n", EmojiLoading, msg)
}

func spaces(n int) string {
	return fmt.Sprintf("%*s", n, "")
}

func (p *Printer) PrintModelMenu(title string, models []string, defaultModels []string) {
	p.PrintTitle(title, EmojiSelect)

	// Determine the maximum length of model names for consistent spacing

	// for _, model := range models {
	// 	if len(model) > maxLen {
	// 		maxLen = len(model)
	// 	}
	// }
	// get the max length of first column model with
	maxLenFirst := 0
	maxLenSecond := 0
	// fist get all model in first column
	firstColumnModels := models[:len(models)/2]
	for _, model := range firstColumnModels {
		if len(model) > maxLenFirst {
			maxLenFirst = len(model)
		}
	}
	// get all model in second column
	secondColumnModels := models[len(models)/2:]
	for _, model := range secondColumnModels {
		if len(model) > maxLenSecond {
			maxLenSecond = len(model)
		}
	}

	// Determine the width needed for numbering based on total models
	numWidth := len(fmt.Sprintf("%d", len(models))) // e.g., 2 for 10 models

	checkMark := "[✓]"
	logger.Debug("checkMark: %d", len(checkMark))

	// Define fixed padding between columns
	colPadding := 4 // Number of spaces between columns

	// Calculate the total width of each column
	// Format: <numWidth>. <model name> <checkMark>
	colWidthFirst := numWidth + 2 + maxLenFirst + 1 + len(checkMark)
	// colWidthSecond := numWidth + 2 + maxLenSecond + 1 + len(checkMark)
	// Calculate number of rows for two columns
	rows := (len(models) + 1) / 2
	for i := 0; i < rows; i++ {
		// First Column
		if i < len(models) {
			// Prepare the format string for the first column
			// %<numWidth>d. %-<maxLen>s <checkMark>
			firstColFormat := fmt.Sprintf("%%%dd. %%-%ds ", numWidth, maxLenFirst)
			p.Printf(firstColFormat, i+1, models[i])

			// Check if it's a default model
			isDefault := false
			for _, defaultModel := range defaultModels {
				if models[i] == defaultModel {
					isDefault = true
					break
				}
			}

			if isDefault {
				p.Printf("%s%s%s", ColorGreen, checkMark, ColorReset)
			} else {
				// Add spaces equivalent to checkMark if not default
				p.Printf("   ")
			}

			// Add padding between columns
			p.Printf("%s", spaces(colPadding))
		} else {
			// If no model exists for the first column in this row, add empty space
			p.Printf("%-*s", colWidthFirst+colPadding, "")
		}

		// Second Column
		secondIndex := i + rows
		if secondIndex < len(models) {
			// Prepare the format string for the second column
			secondColFormat := fmt.Sprintf("%%%dd. %%-%ds ", numWidth, maxLenSecond)
			p.Printf(secondColFormat, secondIndex+1, models[secondIndex])

			// Check if it's a default model
			isDefault := false
			for _, defaultModel := range defaultModels {
				if models[secondIndex] == defaultModel {
					isDefault = true
					break
				}
			}

			if isDefault {
				p.Printf("%s%s%s", ColorGreen, checkMark, ColorReset)
			} else {
				// Add spaces equivalent to checkMark if not default
				p.Printf("%s", "   ") // 3 spaces for "[✓]"
			}
		}

		// Move to the next line after printing both columns
		p.Printf("\n")
	}

	p.Printf("\n%s输入数字选择模型，或填写自定义模型名称%s", ColorGray, ColorReset)
	p.Printf("\n%s多个选项用空格分隔，直接回车使用默认模型%s", ColorGray, ColorReset)
	p.Printf("\n%s[✓] %s为默认模型%s\n", ColorGreen, ColorGray, ColorReset)
	p.Printf("\n%s请选择: %s", ColorBold, ColorReset)
}

// PrintResults prints the test results
func (p *Printer) PrintResults(results interface{}) {
	if results == nil {
		p.PrintSuccess("测试完成")
		return
	}
	p.PrintTitle("测试结果", EmojiDone)
	p.Printf("%+v\n", results)
}
