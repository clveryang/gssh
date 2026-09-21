// Package ui implements the interactive host picker.
package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clveryang/gssh/internal/model"
	"github.com/clveryang/gssh/internal/mru"
)

// Colours are adaptive so the picker stays readable on light and dark terminals.
var (
	styleSelected = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#ffffff"}).
			Background(lipgloss.AdaptiveColor{Light: "#d7e3ff", Dark: "#2d4263"})
	styleName  = lipgloss.NewStyle().Bold(true)
	styleAddr  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#555555", Dark: "#9a9a9a"})
	styleTag   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#8250df", Dark: "#c29fff"})
	styleNote  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#3a7d44", Dark: "#87d096"})
	styleHelp  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#777777", Dark: "#7a7a7a"})
	styleEmpty = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#b3261e", Dark: "#f2b8b5"})
)

// Action is what the user chose to do with the selected host.
type Action int

const (
	ActionNone Action = iota
	ActionConnect
	ActionEdit
)

// Result is what Run returns.
type Result struct {
	Host   *model.Host
	Action Action
}

type item struct {
	host  *model.Host
	score int
	rank  int64
}

type pickerModel struct {
	all      []*model.Host
	shown    []item
	input    textinput.Model
	cursor   int
	height   int
	width    int
	recent   mru.List
	result   Result
	nameWide int
}

// Run shows the picker and blocks until the user picks or quits.
func Run(hosts []*model.Host) (Result, error) {
	ti := textinput.New()
	ti.Placeholder = "type to filter (pinyin works: hzbfj)"
	ti.Prompt = "> "
	ti.Focus()

	m := &pickerModel{
		all:    hosts,
		input:  ti,
		height: 20,
		width:  80,
		recent: mru.Load(),
	}
	m.filter()

	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return Result{}, err
	}
	return final.(*pickerModel).result, nil
}

func (m *pickerModel) Init() tea.Cmd { return textinput.Blink }

func (m *pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if h := m.current(); h != nil {
				m.result = Result{Host: h, Action: ActionConnect}
			}
			return m, tea.Quit
		case "ctrl+e":
			if h := m.current(); h != nil {
				m.result = Result{Host: h, Action: ActionEdit}
			}
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.shown)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	before := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != before {
		m.filter()
	}
	return m, cmd
}

func (m *pickerModel) current() *model.Host {
	if m.cursor < 0 || m.cursor >= len(m.shown) {
		return nil
	}
	return m.shown[m.cursor].host
}

// filter rebuilds the visible list. With no query the order is most-recently-used
// first, which is the whole reason the picker beats shell completion on a long
// list. With a query, better matches win and recency only breaks ties.
func (m *pickerModel) filter() {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	m.shown = m.shown[:0]
	m.nameWide = 0

	for _, h := range m.all {
		s := 0
		if q != "" {
			s = score(h, q)
			if s == 0 {
				continue
			}
		}
		if w := lipgloss.Width(h.Name); w > m.nameWide {
			m.nameWide = w
		}
		m.shown = append(m.shown, item{host: h, score: s, rank: m.recent.Rank(h.Name)})
	}

	sort.SliceStable(m.shown, func(i, j int) bool {
		if m.shown[i].score != m.shown[j].score {
			return m.shown[i].score > m.shown[j].score
		}
		if m.shown[i].rank != m.shown[j].rank {
			return m.shown[i].rank > m.shown[j].rank
		}
		return m.shown[i].host.Name < m.shown[j].host.Name
	})

	if m.cursor >= len(m.shown) {
		m.cursor = max(0, len(m.shown)-1)
	}
}

// score ranks a host against the query. Exact beats prefix beats substring, and
// a hit on the name outranks a hit on a tag or note.
func score(h *model.Host, q string) int {
	best := 0
	for i, hay := range h.Search {
		if hay == "" {
			continue
		}
		var s int
		switch {
		case hay == q:
			s = 100
		case strings.HasPrefix(hay, q):
			s = 60
		case strings.Contains(hay, q):
			s = 30
		default:
			continue
		}
		// Search[0] is the name; later entries (aliases, pinyin, notes) rank lower.
		if i > 0 {
			s -= 5
		}
		if s > best {
			best = s
		}
	}
	return best
}

func (m *pickerModel) View() string {
	var b strings.Builder
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	if len(m.shown) == 0 {
		b.WriteString(styleEmpty.Render("  no host matches"))
		b.WriteString("\n\n")
		b.WriteString(styleHelp.Render("  esc quit"))
		return b.String()
	}

	// Leave room for the input, blank lines and the help footer.
	visible := m.height - 5
	if visible < 3 {
		visible = 3
	}
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := min(start+visible, len(m.shown))

	for i := start; i < end; i++ {
		b.WriteString(m.renderRow(m.shown[i], i == m.cursor))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render(fmt.Sprintf(
		"  %d/%d  ↑↓ move  ⏎ connect  ^e edit  esc quit",
		m.cursor+1, len(m.shown))))
	return b.String()
}

func (m *pickerModel) renderRow(it item, selected bool) string {
	h := it.host

	addr := h.Host
	if h.User != "" {
		addr = h.User + "@" + addr
	}
	if h.Port != 0 {
		addr = fmt.Sprintf("%s:%d", addr, h.Port)
	}

	name := h.Name + strings.Repeat(" ", max(0, m.nameWide-lipgloss.Width(h.Name)))

	var parts []string
	parts = append(parts, styleName.Render(name), styleAddr.Render(addr))
	if len(h.Tags) > 0 {
		parts = append(parts, styleTag.Render("["+strings.Join(h.Tags, ",")+"]"))
	}
	if h.Note != "" {
		parts = append(parts, styleNote.Render(h.Note))
	}
	line := "  " + strings.Join(parts, "  ")

	if selected {
		// Re-render without inner colours so the highlight reads cleanly.
		plain := "› " + name + "  " + addr
		if len(h.Tags) > 0 {
			plain += "  [" + strings.Join(h.Tags, ",") + "]"
		}
		if h.Note != "" {
			plain += "  " + h.Note
		}
		w := m.width
		if w > 2 {
			plain += strings.Repeat(" ", max(0, w-lipgloss.Width(plain)-1))
		}
		return styleSelected.Render(plain)
	}
	return line
}
