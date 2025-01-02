package trace

import (
	"fmt"
	"strings"

	"github.com/go-coders/check-trace/pkg/types"
	"github.com/go-coders/check-trace/pkg/util"
	"github.com/mattn/go-runewidth"
)

var nodeSpace = "   节点"

// formatNodeInfo formats node information for display
func formatNodeInfo(nodeIndex int, node *types.Node) string {
	// Use node fields if they exist, otherwise use info fields
	country := node.Country
	regionName := node.RegionName

	displayWidth := runewidth.StringWidth(node.ServerName)
	padding := 20 - displayWidth
	serviceText := node.ServerName
	if padding > 0 {
		serviceText += strings.Repeat(" ", padding)
	}

	// add color green
	return fmt.Sprintf("%s%s%d : %sIP: %s (%s,%s - %s)%s",
		util.ColorGreen, nodeSpace, nodeIndex, serviceText, node.IP, regionName, country, node.Org, util.ColorReset)
}

func formatNodeRequestCounts(nodes []types.Node) string {

	var result string

	result = "\n节点请求次数："
	for _, node := range nodes {
		result += "\n"
		result += fmt.Sprintf("%s%s%d: %d次%s", util.ColorGreen, nodeSpace, node.NodeIndex, node.RequestCount, util.ColorReset)
	}
	return result
}
