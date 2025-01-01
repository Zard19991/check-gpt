package trace

import (
	"fmt"
	"strings"

	"github.com/go-coders/check-trace/pkg/ipinfo"
	"github.com/go-coders/check-trace/pkg/types"
	"github.com/go-coders/check-trace/pkg/util"
	"github.com/mattn/go-runewidth"
)

var nodeSpace = "   节点"

// formatNodeInfo formats node information for display
func formatNodeInfo(nodeIndex int, node *types.Node, provider ipinfo.Provider) string {
	info, err := provider.GetIPInfo(node.IP)
	if err != nil {
		return fmt.Sprintf("   节点%d : 未知服务IP: %s", nodeIndex, node.IP)
	}

	// Use node fields if they exist, otherwise use info fields
	country := node.Country
	if country == "" {
		country = info.Country
	}

	regionName := node.RegionName
	if regionName == "" {
		regionName = info.RegionName
	}

	org := node.Org
	if org == "" {
		org = info.Org
	}

	// get server from util.GetServer
	service := util.GetPlatformInfo(node.UserAgent)
	// 确保服务名称部分至少有15个字符宽度
	serviceText := fmt.Sprintf("%s服务", service)
	displayWidth := runewidth.StringWidth(serviceText)
	padding := 15 - displayWidth
	if padding > 0 {
		serviceText += strings.Repeat(" ", padding)
	}

	// add country
	return fmt.Sprintf("%s%d : %sIP: %s (%s,%s - %s)",
		nodeSpace, nodeIndex, serviceText, node.IP, regionName, country, org)
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
