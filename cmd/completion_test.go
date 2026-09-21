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
  - name: web-01
    host: 0.0.0.1
  - name: tower
    host: 0.0.0.2
  - name: Staging
    host: 0.0.0.3
  - name: staging
    host: 0.0.0.4
  - name: 杭州备份机
    host: 0.0.0.57
  - name: "42"
    host: 0.0.0.6
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
		{"we", []string{"web-01"}, "prefix beats tower, which only contains we"},
		{"Stag", []string{"Staging"}, "case-sensitive match first, or compadd -U erases the input"},
		{"stag", []string{"staging"}, "same, other case"},
		{"STAG", []string{"Staging", "staging"}, "no exact-case match: fall back to ignoring case"},
		{"hzbfj", []string{"杭州备份机"}, "pinyin initials"},
		{"hangzhou", []string{"杭州备份机"}, "pinyin prefix"},
		{"0.0.0.57", []string{"杭州备份机"}, "IP fragment, last resort"},
		{"4", []string{"42"}, "name prefix wins over IPs that contain a 4"},
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

// cobra's bash script needs the bash-completion package and fails silently
// without it (macOS, minimal servers); ours must stand alone.
func TestBashCompletionIsSelfContained(t *testing.T) {
	var sb strings.Builder
	completionCmd.SetOut(&sb)
	if err := completionCmd.RunE(completionCmd, []string{"bash"}); err != nil {
		t.Fatal(err)
	}
	script := sb.String()
	if strings.Contains(script, "_get_comp_words_by_ref") || strings.Contains(script, "_init_completion") {
		t.Error("bash script depends on the bash-completion package")
	}
	if !strings.Contains(script, "complete -F _gssh_complete gssh") {
		t.Error("bash script does not register a completion function")
	}
}
