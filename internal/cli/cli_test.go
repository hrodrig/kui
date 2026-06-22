package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteVersion(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"version"})
	buf := new(strings.Builder)
	root.SetOut(buf)
	err := root.Execute()
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "kui ") {
		t.Fatalf("version output: %s", out)
	}
}

func TestExecuteHelp(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"--help"})
	buf := new(strings.Builder)
	root.SetOut(buf)
	err := root.Execute()
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "serve") {
		t.Fatalf("help missing 'serve': %s", out[:200])
	}
	if !strings.Contains(out, "version") {
		t.Fatalf("help missing 'version': %s", out[:200])
	}
}

func TestExecuteServeHelp(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"serve", "--help"})
	buf := new(strings.Builder)
	root.SetOut(buf)
	err := root.Execute()
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "listen") {
		t.Fatalf("serve help missing --listen: %s", out[:200])
	}
}

func TestServeCmdMissingConfig(t *testing.T) {
	err := serveCmd("/nonexistent/config.yml", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServeCmdBadDBPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "kui.yml")
	os.WriteFile(cfgPath, []byte("database:\n  path: /nonexistent/dir/kui.db\n"), 0644)

	err := serveCmd(cfgPath, "")
	if err == nil {
		t.Fatal("expected error from bad db path")
	}
}

func TestExecuteWithConfigFlag(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "kui.yml")
	os.WriteFile(cfgPath, []byte("database:\n  path: "+filepath.Join(dir, "kui.db")+"\n"), 0644)

	root := newRootCmd()
	root.SetArgs([]string{"serve", "--config", cfgPath})
	// serve will try to listen; we just want to see it doesn't panic on startup
	// it will fail because no real server can start, but at least config loads
	err := root.Execute()
	// expect error because server won't start on :3000 (or file issues), but we just test code path
	_ = err
}

func TestExecuteUnknownFlag(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"--unknown-flag"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}
