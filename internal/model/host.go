// Package model defines the data structures that make up a gssh configuration.
package model

import "strings"

// Config is the root of ~/.config/gssh/hosts.yaml.
type Config struct {
	Defaults Options `yaml:"defaults,omitempty"`
	Groups   []Group `yaml:"groups,omitempty"`
	Hosts    []*Host `yaml:"hosts,omitempty"` // ungrouped hosts
}

// Group is a named bucket of hosts. Tags set here are inherited by members.
type Group struct {
	Name  string   `yaml:"name"`
	Tags  []string `yaml:"tags,omitempty"`
	Hosts []*Host  `yaml:"hosts"`
}

// Host is a single connectable machine.
type Host struct {
	Name    string   `yaml:"name"`
	Alias   []string `yaml:"alias,omitempty"`
	Host    string   `yaml:"host"`
	Note    string   `yaml:"note,omitempty"`
	Tags    []string `yaml:"tags,omitempty"`
	Pinyin  string   `yaml:"pinyin,omitempty"` // override for auto-derived pinyin
	Options `yaml:",inline"`

	// Populated at load time, never serialised.
	Group  string   `yaml:"-"`
	Search []string `yaml:"-"` // lowercase haystacks: name, aliases, pinyin, host, note
	// SSHName is what gssh passes to ssh; see config.assignSSHNames.
	SSHName string `yaml:"-"`
}

// Options are the ssh_config keywords gssh knows how to render. A zero value
// means "not set" and is omitted from the generated file.
type Options struct {
	User                string   `yaml:"user,omitempty"`
	Port                int      `yaml:"port,omitempty"`
	IdentityFile        string   `yaml:"identity_file,omitempty"`
	ProxyJump           string   `yaml:"proxy_jump,omitempty"`
	ForwardAgent        *bool    `yaml:"forward_agent,omitempty"`
	ServerAliveInterval int      `yaml:"server_alive_interval,omitempty"`
	ServerAliveCountMax int      `yaml:"server_alive_count_max,omitempty"`
	LocalForward        []string `yaml:"local_forward,omitempty"`
	RemoteForward       []string `yaml:"remote_forward,omitempty"`
	DynamicForward      []string `yaml:"dynamic_forward,omitempty"`

	// Raw passes through ssh_config keywords gssh has no typed field for.
	Raw map[string]string `yaml:"raw,omitempty"`
}

// AllHosts flattens groups and loose hosts into one ordered slice.
func (c *Config) AllHosts() []*Host {
	out := make([]*Host, 0, len(c.Hosts))
	for _, g := range c.Groups {
		out = append(out, g.Hosts...)
	}
	return append(out, c.Hosts...)
}

// SSHSafe reports whether ssh accepts s as a host argument. OpenSSH refuses
// non-ASCII and shell metacharacters with "hostname contains invalid
// characters" -- which rules out every Chinese host name.
func SSHSafe(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r <= ' ' || r > '~' || strings.ContainsRune("'`\"$\\;&<>|(){},", r) {
			return false
		}
	}
	return true
}

// Target is the name to hand to ssh for this host.
func (h *Host) Target() string {
	if h.SSHName != "" {
		return h.SSHName
	}
	return h.Name
}
