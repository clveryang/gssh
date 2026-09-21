package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/clveryang/gssh/internal/model"
	"github.com/clveryang/gssh/internal/mru"
)

// hostsFixture mirrors what config.Load would produce, including the lowercase
// Search keys the picker matches against.
func hostsFixture() []*model.Host {
	mk := func(name, addr, note string, search ...string) *model.Host {
		h := &model.Host{Name: name, Host: addr, Note: note}
		h.Search = append([]string{strings.ToLower(name), strings.ToLower(addr)}, search...)
		return h
	}
	return []*model.Host{
		mk("杭州备份机", "10.0.2.104", "backup", "hangzhoubeifenji", "hzbfj"),
		mk("board1", "10.0.1.13", "lab box"),
		mk("board2", "10.0.1.14", ""),
		mk("zgocloud", "64.0.0.1", ""),
	}
}

func newTestPicker(t *testing.T, recent mru.List) *pickerModel {
	t.Helper()
	ti := textinput.New()
	ti.Focus()
	m := &pickerModel{all: hostsFixture(), input: ti, height: 20, width: 80, recent: recent, st: newStyles()}
	m.filter()
	return m
}

func names(m *pickerModel) []string {
	out := make([]string, len(m.shown))
	for i, it := range m.shown {
		out[i] = it.host.Name
	}
	return out
}

func typeQuery(m *pickerModel, q string) {
	for _, r := range q {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestPinyinFilter(t *testing.T) {
	m := newTestPicker(t, mru.List{})
	typeQuery(m, "hzbfj")
	if got := names(m); len(got) != 1 || got[0] != "杭州备份机" {
		t.Errorf("pinyin query gave %v", got)
	}
}

// The reason the picker exists: with no query, recently used hosts come first.
func TestMRUOrdersEmptyQuery(t *testing.T) {
	m := newTestPicker(t, mru.List{"zgocloud": 5000, "board2": 9000})
	got := names(m)
	if got[0] != "board2" || got[1] != "zgocloud" {
		t.Errorf("MRU order wrong: %v", got)
	}
}

// With a query, match quality wins and recency is only a tiebreaker.
func TestScoreBeatsRecency(t *testing.T) {
	m := newTestPicker(t, mru.List{"board2": 9999})
	typeQuery(m, "board1")
	if got := names(m); got[0] != "board1" {
		t.Errorf("exact match should outrank a recent host: %v", got)
	}
}

func TestScoreOrdering(t *testing.T) {
	h := &model.Host{Name: "board1", Search: []string{"board1"}}
	exact, prefix, sub := score(h, "board1"), score(h, "board"), score(h, "oard")
	if !(exact > prefix && prefix > sub && sub > 0) {
		t.Errorf("exact=%d prefix=%d substring=%d", exact, prefix, sub)
	}
	if score(h, "nope") != 0 {
		t.Error("non-match should score 0")
	}
}

func TestNameHitBeatsNoteHit(t *testing.T) {
	name := &model.Host{Name: "lab", Search: []string{"lab"}}
	note := &model.Host{Name: "other", Search: []string{"other", "lab box"}}
	if score(name, "lab") <= score(note, "lab") {
		t.Error("a hit on the name should outrank a hit on the note")
	}
}

func TestCursorStaysInBounds(t *testing.T) {
	m := newTestPicker(t, mru.List{})
	for i := 0; i < 20; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.cursor != len(m.shown)-1 {
		t.Errorf("cursor ran past the end: %d of %d", m.cursor, len(m.shown))
	}
	for i := 0; i < 20; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	if m.cursor != 0 {
		t.Errorf("cursor ran before the start: %d", m.cursor)
	}
}

// Narrowing the list must not leave the cursor pointing at a removed row.
func TestCursorClampsWhenListShrinks(t *testing.T) {
	m := newTestPicker(t, mru.List{})
	for i := 0; i < 3; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	typeQuery(m, "hzbfj")
	if m.cursor >= len(m.shown) {
		t.Fatalf("cursor %d out of range for %d rows", m.cursor, len(m.shown))
	}
	if m.current() == nil {
		t.Error("current() returned nil after the list shrank")
	}
}

func TestEnterSelects(t *testing.T) {
	m := newTestPicker(t, mru.List{})
	typeQuery(m, "zgocloud")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.result.Action != ActionConnect || m.result.Host == nil || m.result.Host.Name != "zgocloud" {
		t.Errorf("enter did not select: %+v", m.result)
	}
}

func TestEscQuitsWithoutSelecting(t *testing.T) {
	m := newTestPicker(t, mru.List{})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.result.Action != ActionNone {
		t.Errorf("esc should not select anything, got %+v", m.result)
	}
}

func TestNoMatchRendersMessage(t *testing.T) {
	m := newTestPicker(t, mru.List{})
	typeQuery(m, "zzzznothing")
	if len(m.shown) != 0 {
		t.Fatalf("expected no matches, got %v", names(m))
	}
	if !strings.Contains(m.View(), "no host matches") {
		t.Error("empty state should say so")
	}
	if m.current() != nil {
		t.Error("current() must be nil with no rows")
	}
}

// A short window must not make the view taller than the terminal.
func TestViewFitsSmallTerminal(t *testing.T) {
	m := newTestPicker(t, mru.List{})
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	lines := strings.Count(m.View(), "\n") + 1
	if lines > 8 {
		t.Errorf("view is %d lines in an 8-line terminal", lines)
	}
}
