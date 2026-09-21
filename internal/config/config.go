// Package config locates, loads and validates the gssh host file.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clveryang/gssh/internal/model"
	"github.com/clveryang/gssh/internal/pyin"
	"gopkg.in/yaml.v3"
)

// Path returns the host file location, honouring GSSH_CONFIG.
func Path() string {
	if p := os.Getenv("GSSH_CONFIG"); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "gssh", "hosts.yaml")
}

// Load reads and indexes the host file. A missing file is not an error; it
// yields an empty config so `gssh import` works on a fresh install.
func Load() (*model.Config, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &model.Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var c model.Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	index(&c)
	return &c, nil
}

// Save writes the config back, creating the directory if needed.
func Save(c *model.Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// index fills in the derived fields every other package relies on.
func index(c *model.Config) {
	for _, g := range c.Groups {
		for _, h := range g.Hosts {
			h.Group = g.Name
			h.Tags = union(g.Tags, h.Tags)
			buildSearch(h)
		}
	}
	for _, h := range c.Hosts {
		buildSearch(h)
	}
}

func buildSearch(h *model.Host) {
	keys := []string{strings.ToLower(h.Name), strings.ToLower(h.Host)}
	for _, a := range h.Alias {
		keys = append(keys, strings.ToLower(a))
	}
	if h.Note != "" {
		keys = append(keys, strings.ToLower(h.Note))
	}
	for _, t := range h.Tags {
		keys = append(keys, strings.ToLower(t))
	}
	if h.Pinyin != "" {
		keys = append(keys, strings.ToLower(h.Pinyin))
	} else {
		keys = append(keys, pyin.Keys(h.Name)...)
		keys = append(keys, pyin.Keys(h.Note)...)
	}
	h.Search = dedupe(keys)
}

// Lookup resolves a name or alias to a host. Matching is case-insensitive
// because ssh_config Host patterns are, and the user's existing config
// already contains both "Tencent" and "tencent".
func Lookup(c *model.Config, name string) (*model.Host, bool) {
	want := strings.ToLower(name)
	for _, h := range c.AllHosts() {
		if strings.ToLower(h.Name) == want {
			return h, true
		}
		for _, a := range h.Alias {
			if strings.ToLower(a) == want {
				return h, true
			}
		}
	}
	return nil, false
}

func union(a, b []string) []string {
	return dedupe(append(append([]string{}, a...), b...))
}

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
