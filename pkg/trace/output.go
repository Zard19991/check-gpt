package trace

import (
	"fmt"

	"github.com/go-coders/check-trace/pkg/util"
)

// defaultOutputWriter provides default implementation of OutputWriter
type defaultOutputWriter struct{}

func (w *defaultOutputWriter) WriteNodeInfo(nodeNum int, info string) {
	fmt.Println(info)
}

func (w *defaultOutputWriter) WriteRequestCounts(counts []string) {
	fmt.Println("\n节点请求统计：")
	for _, count := range counts {
		fmt.Println(count)
	}
}

func (w *defaultOutputWriter) Write(content string) {
	fmt.Println(content)
}

func (w *defaultOutputWriter) WriteInfo(content string) {
	// with color green
	fmt.Printf("%s%s%s", util.ColorGreen, content, util.ColorReset)
}

func (w *defaultOutputWriter) WriteError(err string) {
	// color red
	fmt.Printf("%s%s%s\n", util.ColorRed, err, util.ColorReset)
}

func (w *defaultOutputWriter) WriteResponse(content string) {
	// color cyan
	fmt.Printf("\n%s%s%s\n", util.ColorCyan, content, util.ColorReset)
}
