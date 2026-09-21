package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clveryang/gssh/internal/model"
)

func TestFragmentMergesDefaults(t *testing.T) {
	c := &model.Config{
		Defaults: model.Options{IdentityFile: "~/.ssh/id_rsa", ServerAliveInterval: 30},
		Hosts: []*model.Host{
			{Name: "a", Host: "1.2.3.4", Options: model.Options{User: "root"}},
			{Name: "b", Host: "5.6.7.8", Options: model.Options{IdentityFile: "~/.ssh/other"}},
		},
	}
	out := Fragment(c, "test.yaml")

	if !strings.Contains(out, "Host a\n    HostName 1.2.3.4\n    User root\n    IdentityFile ~/.ssh/id_rsa") {
		t.Errorf("host a did not inherit defaults:\n%s", out)
	}
	if !strings.Contains(out, "IdentityFile ~/.ssh/other") {
		t.Error("host b should override the default IdentityFile")
	}
	if strings.Count(out, "ServerAliveInterval 30") != 2 {
		t.Error("both hosts should inherit ServerAliveInterval")
	}
}

func TestFragmentAliases(t *testing.T) {
	c := &model.Config{Hosts: []*model.Host{
		{Name: "shanghai", Alias: []string{"sh", "13"}, Host: "10.0.0.1", Note: "lab box"},
	}}
	out := Fragment(c, "t")
	if !strings.Contains(out, "Host shanghai sh 13") {
		t.Errorf("aliases not rendered:\n%s", out)
	}
	if !strings.Contains(out, "# lab box") {
		t.Error("note should be emitted as a comment")
	}
}

// EnsureInclude must land after leading Include lines (another tool owns them)
// but before the first Host block, so gssh wins over stale hand-written dupes.
func TestEnsureIncludePlacement(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	os.WriteFile(cfg, []byte("Include /other/tool.conf\n\nHost legacy\n    HostName 9.9.9.9\n"), 0o600)

	changed, err := EnsureInclude(cfg, "/frag.conf")
	if err != nil || !changed {
		t.Fatalf("EnsureInclude: changed=%v err=%v", changed, err)
	}
	got, _ := os.ReadFile(cfg)
	body := string(got)

	iOther := strings.Index(body, "/other/tool.conf")
	iGssh := strings.Index(body, "/frag.conf")
	iHost := strings.Index(body, "Host legacy")
	if !(iOther < iGssh && iGssh < iHost) {
		t.Errorf("wrong placement (other=%d gssh=%d host=%d):\n%s", iOther, iGssh, iHost, body)
	}
}

func TestEnsureIncludeIdempotent(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	os.WriteFile(cfg, []byte("Host x\n    HostName 1.1.1.1\n"), 0o600)

	if changed, _ := EnsureInclude(cfg, "/frag.conf"); !changed {
		t.Fatal("first call should modify")
	}
	if changed, _ := EnsureInclude(cfg, "/frag.conf"); changed {
		t.Error("second call should be a no-op")
	}
	got, _ := os.ReadFile(cfg)
	if n := strings.Count(string(got), BeginMarker); n != 1 {
		t.Errorf("marker written %d times, want 1", n)
	}
}

func TestEnsureIncludeEmptyFile(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config") // does not exist
	if _, err := EnsureInclude(cfg, "/frag.conf"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg); err != nil {
		t.Fatal("config should have been created")
	}
}
