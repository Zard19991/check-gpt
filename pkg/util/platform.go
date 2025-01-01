package util

import "strings"

const platformUnknown = "未知"

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
func GetPlatformInfo(userAgent string) string {
	// Return Unknown for empty user agent
	if userAgent == "" {
		return platformUnknown
	}

	for _, platform := range platformPatterns {
		for _, pattern := range platform.patterns {
			if platform.caseSensitive {
				if strings.Contains(userAgent, pattern) {
					return platform.name
				}
			} else {
				if strings.Contains(strings.ToLower(userAgent), strings.ToLower(pattern)) {
					return platform.name
				}
			}
		}
	}

	// Return original user agent if no pattern matches
	return userAgent
}
