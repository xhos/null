package version

import "fmt"

const (
	RepoName = "ariand"
	RepoURL  = "https://github.com/xhos/ariand"
)

var (
	Version   = "dev"
	GitCommit = "unknown"
)

func Full() string {
	return fmt.Sprintf("%s (%s)", Version, GitCommit[:7])
}

func Short() string {
	return Version
}
