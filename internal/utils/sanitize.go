// internal/utils/sanitize.go
package utils

import (
	"fmt"
	"regexp"
)

var (
	youtubeRegex = regexp.MustCompile(`(?:youtube\.com/watch\?v=|youtu\.be/)([^&\s]+)`)
	vimeoRegex   = regexp.MustCompile(`vimeo\.com/(\d+)`)
)

func SanitizeVideoEmbed(input string) string {
	if match := youtubeRegex.FindStringSubmatch(input); match != nil {
		return fmt.Sprintf(`<iframe src="https://www.youtube.com/embed/%s"></iframe>`, match[1])
	}
	if match := vimeoRegex.FindStringSubmatch(input); match != nil {
		return fmt.Sprintf(`<iframe src="https://player.vimeo.com/video/%s"></iframe>`, match[1])
	}
	return ""
}
