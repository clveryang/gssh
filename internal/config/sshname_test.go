package config

import (
	"testing"

	"github.com/clveryang/gssh/internal/model"
)

// ssh rejects non-ASCII host arguments ("hostname contains invalid
// characters"), so every host needs a name ssh will take.
func TestAssignSSHNames(t *testing.T) {
	hosts := []*model.Host{
		{Name: "web-01"},
		{Name: "杭州备份机"},                        // pinyin
		{Name: "北京GPU", Alias: []string{"bj"}}, // an ASCII alias wins
		{Name: "广州生产1"},
		{Name: "广州生产1 "}, // same pinyin -> must not collide
		{Name: "hangzhoubeifenji-x", Alias: []string{"hangzhoubeifenji"}},
	}
	assignSSHNames(hosts)

	want := map[int]string{0: "web-01", 2: "bj", 3: "guangzhoushengchan1"}
	for i, w := range want {
		if got := hosts[i].Target(); got != w {
			t.Errorf("%q: Target() = %q, want %q", hosts[i].Name, got, w)
		}
	}
	// 杭州备份机's pinyin is already taken by another host's alias.
	if got := hosts[1].Target(); got == "hangzhoubeifenji" || !model.SSHSafe(got) {
		t.Errorf("杭州备份机 got %q: must be ssh-safe and not reuse a taken name", got)
	}
	if hosts[3].Target() == hosts[4].Target() {
		t.Errorf("two hosts share ssh name %q", hosts[3].Target())
	}
	for _, h := range hosts {
		if !model.SSHSafe(h.Target()) {
			t.Errorf("%q: %q is not ssh-safe", h.Name, h.Target())
		}
	}
}
