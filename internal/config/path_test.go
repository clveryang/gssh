package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// On macOS os.UserConfigDir() is ~/Library/Application Support, which is not
// where a CLI's config belongs and is not what the docs promise.
func TestPathUsesDotConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GSSH_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	got := Path()
	want := filepath.Join(home, ".config", "gssh", "hosts.yaml")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
	if strings.Contains(got, "Application Support") {
		t.Error("config must not live in ~/Library/Application Support")
	}
}

func TestPathHonoursXDG(t *testing.T) {
	home := t.TempDir()
	xdg := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GSSH_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	if got, want := Path(), filepath.Join(xdg, "gssh", "hosts.yaml"); got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestGSSHConfigWins(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GSSH_CONFIG", "/explicit/path.yaml")

	if got := Path(); got != "/explicit/path.yaml" {
		t.Errorf("GSSH_CONFIG should win, got %q", got)
	}
}

// An upgrade must not lose hosts someone already has in the old location.
func TestLegacyPathStillRead(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GSSH_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	legacy := legacyPath()
	if legacy == "" || legacy == xdgPath() {
		t.Skip("no distinct legacy location on this platform")
	}
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("hosts: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := Path(); got != legacy {
		t.Errorf("existing legacy file should still be read: got %q, want %q", got, legacy)
	}

	// Once the new location exists it wins.
	preferred := xdgPath()
	os.MkdirAll(filepath.Dir(preferred), 0o700)
	os.WriteFile(preferred, []byte("hosts: []\n"), 0o600)
	if got := Path(); got != preferred {
		t.Errorf("new location should win once present: got %q", got)
	}
}
