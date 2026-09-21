package cmd

import (
	"strings"
	"testing"

	"github.com/clveryang/gssh/internal/model"
)

func TestShadowedBy(t *testing.T) {
	cases := map[string]string{
		"ls":     "ls",
		"list":   "ls",   // alias
		"write":  "edit", // alias
		"help":   "help", // added lazily by cobra
		"doctor": "doctor",
		"LS":     "", // cobra matching is case-sensitive, so not shadowed
		"web":    "",
	}
	for name, want := range cases {
		if got := shadowedBy(name); got != want {
			t.Errorf("shadowedBy(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestDoctorReportsShadowedHost(t *testing.T) {
	c := &model.Config{Hosts: []*model.Host{
		{Name: "ls", Host: "10.0.0.9", Note: "x"},
		{Name: "web", Host: "10.0.0.1", Note: "x", Alias: []string{"sync"}},
	}}
	var msgs []string
	for _, p := range lint(c) {
		msgs = append(msgs, p.Host+": "+p.Msg)
	}
	all := strings.Join(msgs, "\n")
	if !strings.Contains(all, "gssh -- ls") {
		t.Errorf("doctor should explain how to reach host ls:\n%s", all)
	}
	if !strings.Contains(all, "alias sync") {
		t.Errorf("doctor should flag the shadowed alias:\n%s", all)
	}
}
