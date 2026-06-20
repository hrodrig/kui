package version

import "testing"

func TestBuildInfoString(t *testing.T) {
	i := Info{
		Version:   "0.1.0",
		Commit:    "abc123",
		BuildDate: "2026-06-20",
		Branch:    "main",
	}
	want := "kui 0.1.0 (commit abc123, built 2026-06-20, branch main)"
	if got := i.String(); got != want {
		t.Errorf("String() = %q; want %q", got, want)
	}
}

func TestBuildInfoUsesPackageVars(t *testing.T) {
	Version = "1.2.3"
	Commit = "deadbeef"
	BuildDate = "today"
	Branch = "develop"

	i := BuildInfo()
	if i.Version != "1.2.3" || i.Commit != "deadbeef" || i.BuildDate != "today" || i.Branch != "develop" {
		t.Fatalf("BuildInfo() = %+v", i)
	}
}
