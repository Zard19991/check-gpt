package menu

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/go-coders/check-gpt/pkg/util"
)

// MenuItem represents a menu item
type MenuItem struct {
	ID    int
	Label string
	Emoji string
}

// ShowMainMenu displays the main menu and returns the user's choice
func ShowMainMenu(input io.Reader, output io.Writer) (MenuItem, error) {
	printer := util.NewPrinter(output)
	printer.PrintTitle("主菜单", util.EmojiRocket)

	items := []MenuItem{
		{ID: 1, Label: "模型测试", Emoji: util.EmojiAPI},
		{ID: 2, Label: "链路检测", Emoji: util.EmojiLink},
		{ID: 3, Label: "检查更新", Emoji: util.EmojiGear},
		{ID: 4, Label: "退出", Emoji: util.EmojiWave},
	}

	for _, item := range items {
		printer.Printf("\n%s %d. %s", item.Emoji, item.ID, item.Label)
	}

	printer.Printf("\n\n%s请选择: %s", util.ColorBold, util.ColorReset)

	reader := bufio.NewReader(input)
	line, err := reader.ReadString('\n')
	if err != nil {
		return MenuItem{}, fmt.Errorf("读取选择失败: %v", err)
	}

	choice := strings.TrimSpace(line)
	id, err := strconv.Atoi(choice)
	if err != nil || id < 1 || id > len(items) {
		return MenuItem{}, fmt.Errorf("无效的选择: %s", choice)
	}

	return items[id-1], nil
}
