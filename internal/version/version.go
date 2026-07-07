package version

import (
	"fmt"
	"runtime"
	"strings"
)

const AppName = "snailproxy"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func Info() string {
	lines := []string{
		fmt.Sprintf("%s %s", AppName, strings.TrimSpace(Version)),
		fmt.Sprintf("commit: %s", strings.TrimSpace(Commit)),
		fmt.Sprintf("built: %s", strings.TrimSpace(BuildDate)),
		fmt.Sprintf("go: %s", runtime.Version()),
		fmt.Sprintf("platform: linux/%s", runtime.GOARCH),
	}
	return strings.Join(lines, "\n")
}
