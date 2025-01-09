package apitest

import (
	"fmt"
	"strings"

	"github.com/go-coders/check-gpt/pkg/config"
	"github.com/go-coders/check-gpt/pkg/util"
)

// PrintModelMenu prints the model selection menu
func PrintModelMenu(p *util.Printer, title string, models []string, defaultModels []string) {
	p.PrintTitle(title, util.EmojiSelect)

	// Print model groups first
	i := 1
	// get model title max length and calculate total width needed
	maxTotalWidth := 0
	for _, group := range config.ModelGroups {
		totalWidth := len(fmt.Sprintf("%d. %s", i, group.Title))
		if totalWidth > maxTotalWidth {
			maxTotalWidth = totalWidth
		}
		i++
	}

	// Reset i for actual printing
	i = 1
	// Add padding for consistent alignment
	for _, group := range config.ModelGroups {
		prefix := fmt.Sprintf("%d. ", i)
		currentWidth := len(prefix) + len(group.Title)
		padding := maxTotalWidth - currentWidth
		p.Printf("\n%s%s%s%s: %s%s",
			util.ColorLightBlue,
			prefix,
			group.Title,
			strings.Repeat(" ", padding),
			strings.Join(group.Models, ", "),
			util.ColorReset)
		i++
	}

	p.Printf("\n") // Add a single blank line

	// Calculate the maximum length for model names
	maxLen := 0
	for _, model := range models {
		if len(model) > maxLen {
			maxLen = len(model)
		}
	}

	// Print individual models in two columns
	numWidth := 2 // Fixed width for numbers
	rows := (len(models) + 1) / 2

	for row := 0; row < rows; row++ {
		// First column
		if row < len(models) {
			format := fmt.Sprintf("%%%dd. %%-%ds", numWidth, maxLen+5) // +5 for spacing between columns
			p.Printf(format, row+i, models[row])                       // Start from i since previous numbers are used by groups
		}

		// Second column
		secondIdx := row + rows
		if secondIdx < len(models) {
			format := fmt.Sprintf("%%%dd. %%s", numWidth)
			p.Printf(format, secondIdx+i, models[secondIdx])
		}
		p.Printf("\n")
	}

	p.Printf("\n%s%s%s", util.ColorLightBlue, config.InputPromptModelDescription, util.ColorReset)
	p.Printf("\n%s%s%s", util.ColorLightBlue, config.InputPromptModelDescription2, util.ColorReset)
	p.Printf("\n%s%s%s", util.ColorLightBlue, config.InputPromptModelDescription3, util.ColorReset)
	p.Printf("\n\n%s请输入: %s", util.ColorBold, util.ColorReset)
}
