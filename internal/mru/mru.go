// Package mru records which hosts were connected to recently, so the picker can
// float the handful of machines you actually use to the top of a long list.
package mru

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// maxEntries caps the file so it cannot grow without bound.
const maxEntries = 200

// List maps host name to last-used unix time.
type List map[string]int64

func path() string {
	if p := os.Getenv("GSSH_MRU"); p != "" {
		return p
	}
	// ~/.cache for the same reason Path uses ~/.config: this is a CLI.
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".cache", "gssh", "recent.json")
		}
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "gssh", "recent.json")
}

// Load reads the list. A missing or corrupt file is not an error: this is a
// convenience cache, and losing it must never block a connection.
func Load() List {
	data, err := os.ReadFile(path())
	if err != nil {
		return List{}
	}
	var l List
	if err := json.Unmarshal(data, &l); err != nil {
		return List{}
	}
	return l
}

// Touch records a connection to name. Errors are ignored for the same reason.
func Touch(name string) {
	l := Load()
	l[name] = time.Now().Unix()

	if len(l) > maxEntries {
		type kv struct {
			k string
			v int64
		}
		all := make([]kv, 0, len(l))
		for k, v := range l {
			all = append(all, kv{k, v})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
		l = List{}
		for _, e := range all[:maxEntries] {
			l[e.k] = e.v
		}
	}

	p := path()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return
	}
	data, err := json.Marshal(l)
	if err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0o600)
}

// Rank returns a sort key: larger is more recent, 0 means never used.
func (l List) Rank(name string) int64 { return l[name] }
