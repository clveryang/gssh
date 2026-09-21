// Package sshconf parses an existing ssh_config well enough to import it.
//
// It deliberately does not follow Include directives: those files are owned by
// other tools (colima, for one) and importing them would fight over them.
package sshconf

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/clveryang/gssh/internal/model"
)

// Result is what a parse yielded.
type Result struct {
	Hosts     []*model.Host
	Wildcards []string // Host patterns skipped because they contain * or ?
	Includes  []string // Include lines seen, reported but not followed
}

// typed maps lowercase ssh_config keywords onto model.Options fields.
func setTyped(o *model.Options, key, val string) bool {
	switch strings.ToLower(key) {
	case "user":
		o.User = val
	case "port":
		if n, err := strconv.Atoi(val); err == nil {
			o.Port = n
		}
	case "identityfile":
		o.IdentityFile = val
	case "proxyjump":
		o.ProxyJump = val
	case "forwardagent":
		b := strings.EqualFold(val, "yes")
		o.ForwardAgent = &b
	case "serveraliveinterval":
		if n, err := strconv.Atoi(val); err == nil {
			o.ServerAliveInterval = n
		}
	case "serveralivecountmax":
		if n, err := strconv.Atoi(val); err == nil {
			o.ServerAliveCountMax = n
		}
	case "localforward":
		o.LocalForward = append(o.LocalForward, val)
	case "remoteforward":
		o.RemoteForward = append(o.RemoteForward, val)
	case "dynamicforward":
		o.DynamicForward = append(o.DynamicForward, val)
	case "hostname":
		return false // handled by the caller
	default:
		return false
	}
	return true
}

// ParseFile reads an ssh_config from disk.
func ParseFile(path string) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse reads an ssh_config from r.
func Parse(r io.Reader) (*Result, error) {
	res := &Result{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var cur *model.Host
	var pendingNote string

	flush := func() {
		if cur != nil {
			res.Hosts = append(res.Hosts, cur)
			cur = nil
		}
	}

	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimSpace(raw)

		if line == "" {
			pendingNote = ""
			continue
		}
		if strings.HasPrefix(line, "#") {
			// Remember a lone comment; it often names the machine.
			pendingNote = strings.TrimSpace(strings.TrimLeft(line, "#"))
			continue
		}

		key, val := splitKV(line)
		if key == "" {
			continue
		}

		switch strings.ToLower(key) {
		case "include":
			res.Includes = append(res.Includes, val)
			continue
		case "match":
			// Match blocks have no stable name; end the current host and skip.
			flush()
			continue
		case "host":
			flush()
			patterns := strings.Fields(val)
			if len(patterns) == 0 {
				continue
			}
			if hasWildcard(patterns) {
				res.Wildcards = append(res.Wildcards, val)
				continue
			}
			cur = &model.Host{Name: patterns[0], Alias: patterns[1:], Note: pendingNote}
			pendingNote = ""
			continue
		}

		if cur == nil {
			continue // keyword outside any Host block
		}
		if strings.EqualFold(key, "hostname") {
			cur.Host = val
			continue
		}
		if setTyped(&cur.Options, key, val) {
			continue
		}
		if cur.Raw == nil {
			cur.Raw = map[string]string{}
		}
		cur.Raw[key] = val
	}
	flush()

	if err := sc.Err(); err != nil {
		return nil, err
	}
	// A Host with no HostName resolves to its own name in ssh; make that explicit.
	for _, h := range res.Hosts {
		if h.Host == "" {
			h.Host = h.Name
		}
	}
	return res, nil
}

func splitKV(line string) (string, string) {
	// ssh_config accepts "Key Value" and "Key=Value".
	if i := strings.IndexAny(line, " \t="); i > 0 {
		return line[:i], strings.TrimSpace(strings.TrimLeft(line[i:], " \t="))
	}
	return line, ""
}

func hasWildcard(patterns []string) bool {
	for _, p := range patterns {
		if strings.ContainsAny(p, "*?!") {
			return true
		}
	}
	return false
}
