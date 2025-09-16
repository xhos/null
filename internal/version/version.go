package version

import (
	"fmt"
	"runtime"
)

const (
	Major    = 0
	Minor    = 2
	Patch    = 0
	RepoName = "ariand"
	RepoURL  = "https://github.com/xhos/ariand"
)

var (
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

func Version() string {
	return fmt.Sprintf("v%d.%d.%d", Major, Minor, Patch)
}

func FullVersion() string {
	return fmt.Sprintf("%s (commit: %s, branch: %s, built: %s, go: %s)",
		Version(), GitCommit, GitBranch, BuildTime, runtime.Version())
}

func Short() string {
	return fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)
}
