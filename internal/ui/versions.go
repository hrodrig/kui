package ui

import (
	"fmt"
	"strings"

	"github.com/hrodrig/kui/internal/version"
)

const (
	KikoRepoURL = "https://github.com/hrodrig/kiko"
	KUIRepoURL  = "https://github.com/hrodrig/kui"
)

var kikoVersion string

func SetKikoVersion(v string) {
	kikoVersion = strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func KikoVersionLabel() string {
	if kikoVersion == "" {
		return ""
	}
	return fmt.Sprintf("kiko v%s", kikoVersion)
}

func KUIVersionLabel() string {
	v := strings.TrimPrefix(strings.TrimSpace(version.BuildInfo().Version), "v")
	if v == "" {
		return ""
	}
	return fmt.Sprintf("kui v%s", v)
}

func ShowVersionTags() bool {
	return KikoVersionLabel() != "" || KUIVersionLabel() != ""
}
