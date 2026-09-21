package sshconf

import (
	"strings"
	"testing"
)

const sample = `Include /other/tool.conf

# the shanghai box
Host shanghai sh
    HostName 10.0.0.1
    User root
    Port 2222
    LocalForward 8080 localhost:8080
    SomeUnknownKeyword value

Host *
    ForwardAgent yes

Host bare
`

func TestParse(t *testing.T) {
	res, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hosts) != 2 {
		t.Fatalf("got %d hosts, want 2", len(res.Hosts))
	}
	if len(res.Includes) != 1 || len(res.Wildcards) != 1 {
		t.Errorf("includes=%v wildcards=%v", res.Includes, res.Wildcards)
	}

	h := res.Hosts[0]
	if h.Name != "shanghai" || len(h.Alias) != 1 || h.Alias[0] != "sh" {
		t.Errorf("name/alias wrong: %q %v", h.Name, h.Alias)
	}
	if h.Host != "10.0.0.1" || h.User != "root" || h.Port != 2222 {
		t.Errorf("fields wrong: %+v", h.Options)
	}
	if h.Note != "the shanghai box" {
		t.Errorf("note = %q, want the preceding comment", h.Note)
	}
	if len(h.LocalForward) != 1 {
		t.Error("LocalForward not captured")
	}
	if h.Raw["SomeUnknownKeyword"] != "value" {
		t.Errorf("unknown keyword should fall through to Raw, got %v", h.Raw)
	}

	// A Host with no HostName resolves to its own name.
	if res.Hosts[1].Host != "bare" {
		t.Errorf("bare host = %q, want %q", res.Hosts[1].Host, "bare")
	}
}

func TestParseEqualsForm(t *testing.T) {
	res, _ := Parse(strings.NewReader("Host a\n  HostName=1.2.3.4\n  Port=22\n"))
	if len(res.Hosts) != 1 || res.Hosts[0].Host != "1.2.3.4" || res.Hosts[0].Port != 22 {
		t.Errorf("Key=Value form not handled: %+v", res.Hosts)
	}
}
