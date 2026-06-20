package version

import "fmt"

// Info is the build metadata exposed by `kui version` and GET /api/v1/version.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	Branch    string `json:"branch"`
}

// BuildInfo returns the current build metadata from ldflags / defaults.
func BuildInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		Branch:    Branch,
	}
}

// String matches the `kui version` CLI output.
func (i Info) String() string {
	return fmt.Sprintf("kui %s (commit %s, built %s, branch %s)",
		i.Version, i.Commit, i.BuildDate, i.Branch)
}
