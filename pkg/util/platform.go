package util

import (
	"fmt"
	"net"
	"strings"
)

// platformPattern defines a platform and its matching patterns
type platformPattern struct {
	name          string
	patterns      []string
	caseSensitive bool // whether to match with case sensitivity
}

// platformPatterns defines the ordered list of platform patterns to check
var platformPatterns = []platformPattern{
	{"Azure", []string{"IPS", "Azure"}, true},
	{"OpenAI", []string{"OpenAI"}, true},

	{"Python", []string{"python", "requests"}, false},
	{"Node.js", []string{"node", "got", "axios", "fetch"}, false},
	{"Go", []string{"go-http", "fasthttp"}, false},
	{"Java", []string{"java", "okhttp"}, false},
	{"PHP", []string{"php", "laravel", "symfony"}, false},
}

// GetPlatformInfo extracts platform information from User-Agent
func GetPlatformInfo(userAgent string, ip string, cidr []string) string {

	for _, cidr := range cidr {
		if IsIPInCidr(ip, cidr) {
			return "OpenAI服务"
		}
	}
	// Return Unknown for empty user agent
	if userAgent == "" {
		return "未知服务"
	}

	for _, platform := range platformPatterns {
		for _, pattern := range platform.patterns {
			if platform.caseSensitive {
				if strings.Contains(userAgent, pattern) {
					name := platform.name
					if name == "OpenAI" {
						name = fmt.Sprintf("可能是%s", name)
					}
					return fmt.Sprintf("%s服务", name)
				}
			} else {
				if strings.Contains(strings.ToLower(userAgent), strings.ToLower(pattern)) {
					name := platform.name
					if name == "OpenAI" {
						name = fmt.Sprintf("可能是%s", name)
					}
					return fmt.Sprintf("%s服务", name)
				}
			}
		}
	}

	// Return original user agent if no pattern matches
	return fmt.Sprintf("未知服务,User-Agent:%s", userAgent)
}

func IsIPInCidr(ip string, cidr string) bool {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ipNet.Contains(net.ParseIP(ip))
}
