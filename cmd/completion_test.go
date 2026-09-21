package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func completionFixture(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hosts.yaml")
	body := `hosts:
  - name: vast.ai.5060
    host: 70.0.0.1
  - name: NvBoard
    host: 10.207.16.33
  - name: Tencent
    host: 10.10.251.118
  - name: tencent
    host: 10.10.251.119
  - name: 美亚镜像
    host: 10.113.2.104
  - name: "13"
    host: 10.112.32.13
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GSSH_CONFIG", path)
}

func complete(t *testing.T, typed string) []string {
	t.Helper()
	got, _ := completeHosts(root, nil, typed)
	names := make([]string, len(got))
	for i, g := range got {
		names[i], _, _ = strings.Cut(g, "\t")
	}
	return names
}

func TestCompletionTiers(t *testing.T) {
	completionFixture(t)
	tests := []struct {
		typed string
		want  []string
		why   string
	}{
		{"v", []string{"vast.ai.5060"}, "prefix beats NvBoard, which only contains a v"},
		{"Tenc", []string{"Tencent"}, "case-sensitive match first, or compadd -U erases the input"},
		{"tenc", []string{"tencent"}, "same, other case"},
		{"TENC", []string{"Tencent", "tencent"}, "no exact-case match: fall back to ignoring case"},
		{"myjx", []string{"美亚镜像"}, "pinyin initials"},
		{"meiya", []string{"美亚镜像"}, "pinyin prefix"},
		{"10.113", []string{"美亚镜像"}, "IP fragment, last resort"},
		{"1", []string{"13"}, "name prefix wins over the many IPs containing 1"},
	}
	for _, tt := range tests {
		if got := complete(t, tt.typed); !slices.Equal(got, tt.want) {
			t.Errorf("%q: got %v, want %v (%s)", tt.typed, got, tt.want, tt.why)
		}
	}
}

func TestCompletionScriptUsesCompaddU(t *testing.T) {
	var sb strings.Builder
	completionCmd.SetOut(&sb)
	if err := completionCmd.RunE(completionCmd, []string{"zsh"}); err != nil {
		t.Fatal(err)
	}
	script := sb.String()
	for _, want := range []string{"_gssh_cobra()", "compadd -U", "compdef _gssh gssh"} {
		if !strings.Contains(script, want) {
			t.Errorf("zsh script missing %q", want)
		}
	}
	if strings.Count(script, "compdef _gssh gssh") != 1 {
		t.Error("cobra's own compdef should have been removed")
	}
}
