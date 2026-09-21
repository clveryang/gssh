package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clveryang/gssh/internal/model"
)

func withConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hosts.yaml")
	if body != "" {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GSSH_CONFIG", path)
	return path
}

// The whole reason Insert walks a node tree instead of marshalling the struct:
// a user's comments must survive.
func TestInsertPreservesComments(t *testing.T) {
	path := withConfig(t, `# top level note the user wrote
defaults:
  user: root   # inline comment

hosts:
  # this box is special
  - name: existing
    host: 1.1.1.1
`)
	if err := Insert(&model.Host{Name: "added", Host: "2.2.2.2"}, ""); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	body := string(got)

	for _, want := range []string{
		"# top level note the user wrote",
		"# inline comment",
		"# this box is special",
		"name: added",
		"name: existing",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("lost %q after insert:\n%s", want, body)
		}
	}
}

func TestInsertIntoNewGroup(t *testing.T) {
	withConfig(t, "hosts: []\n")
	if err := Insert(&model.Host{Name: "a", Host: "1.1.1.1"}, "lab"); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Groups) != 1 || c.Groups[0].Name != "lab" {
		t.Fatalf("group not created: %+v", c.Groups)
	}
	if len(c.Groups[0].Hosts) != 1 || c.Groups[0].Hosts[0].Name != "a" {
		t.Errorf("host not in group: %+v", c.Groups[0].Hosts)
	}
}

func TestInsertIntoExistingGroup(t *testing.T) {
	withConfig(t, `groups:
  - name: lab
    tags: [x]
    hosts:
      - name: first
        host: 1.1.1.1
`)
	if err := Insert(&model.Host{Name: "second", Host: "2.2.2.2"}, "lab"); err != nil {
		t.Fatal(err)
	}
	c, _ := Load()
	if len(c.Groups) != 1 || len(c.Groups[0].Hosts) != 2 {
		t.Fatalf("want one group with two hosts, got %+v", c.Groups)
	}
	// Group tags must still be inherited by both.
	for _, h := range c.Groups[0].Hosts {
		if len(h.Tags) == 0 || h.Tags[0] != "x" {
			t.Errorf("%s lost the group tag: %v", h.Name, h.Tags)
		}
	}
}

// A missing file must not crash: add is a reasonable first command to run.
func TestInsertIntoMissingFile(t *testing.T) {
	withConfig(t, "")
	if err := Insert(&model.Host{Name: "a", Host: "1.1.1.1"}, ""); err != nil {
		t.Fatal(err)
	}
	c, _ := Load()
	if len(c.AllHosts()) != 1 {
		t.Errorf("want 1 host, got %d", len(c.AllHosts()))
	}
}
