// Package tui is a presentation layer over the same operations as the CLI.
package tui

import (
	tea "charm.land/bubbletea/v2"
	lip "charm.land/lipgloss/v2"
	"context"
	"fmt"
	"github.com/DeepakSilaych/sess/internal/api"
	"github.com/DeepakSilaych/sess/internal/transport"
	"github.com/charmbracelet/x/ansi"
	"os"
	"strings"
	"time"
	"unicode"
)

type Service interface {
	List(context.Context) ([]api.Session, error)
	Create(context.Context, string) (api.Session, error)
	Remove(context.Context, api.Session) error
}
type Choice struct {
	Host    string
	Session *api.Session
}
type Model struct {
	host                            string
	service                         Service
	ctx                             context.Context
	sessions                        []api.Session
	cursor, width, height, frame    int
	filter, input, mode             string
	loading, mutating               bool
	failure, notice, operationError string
	pending                         *api.Session
	updated                         time.Time
	choice                          Choice
	generation                      int
}
type listMsg struct {
	sessions   []api.Session
	err        error
	generation int
}
type createMsg struct {
	session api.Session
	err     error
}
type removeMsg struct {
	err  error
	name string
}
type tickMsg time.Time

func New(ctx context.Context, host string, s Service) Model {
	return Model{ctx: ctx, host: host, service: s, width: 90, height: 28, loading: true}
}
func (m Model) Init() tea.Cmd { return tea.Batch(m.fetch(), tick()) }
func tick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}
func (m Model) fetch() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 12*time.Second)
		defer cancel()
		ss, e := m.service.List(ctx)
		return listMsg{ss, e, m.generation}
	}
}
func (m Model) visible() []api.Session {
	ss := []api.Session{}
	for _, s := range m.sessions {
		if strings.Contains(strings.ToLower(s.Name), strings.ToLower(m.filter)) {
			ss = append(ss, s)
		}
	}
	return ss
}
func (m Model) selected() *api.Session {
	ss := m.visible()
	if len(ss) == 0 {
		return nil
	}
	s := ss[min(m.cursor, len(ss)-1)]
	return &s
}
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
	case tickMsg:
		m.frame++
		if m.frame%20 == 0 && !m.loading && !m.mutating && m.mode == "" {
			m.loading = true
			return m, tea.Batch(m.fetch(), tick())
		}
		return m, tick()
	case listMsg:
		if v.generation != m.generation {
			return m, nil
		}
		m.loading = false
		if v.err != nil {
			m.failure = v.err.Error()
		} else {
			selected := ""
			if s := m.selected(); s != nil {
				selected = s.ID
			}
			m.sessions = v.sessions
			m.failure = ""
			m.updated = time.Now()
			m.cursor = 0
			for i, s := range m.visible() {
				if s.ID == selected {
					m.cursor = i
				}
			}
		}
	case createMsg:
		m.mutating = false
		if v.err != nil {
			m.operationError = v.err.Error()
			m.mode = ""
		} else {
			m.choice = Choice{m.host, &v.session}
			return m, tea.Quit
		}
	case removeMsg:
		m.mutating = false
		m.mode = ""
		m.input = ""
		if v.err != nil {
			m.operationError = v.err.Error()
		} else {
			m.operationError = ""
			m.notice = "Removed " + v.name
		}
		m.loading = true
		return m, m.fetch()
	case tea.KeyPressMsg:
		key := v.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.mutating {
			return m, nil
		}
		if m.mode != "" {
			if key == "esc" {
				m.mode = ""
				m.input = ""
				return m, nil
			}
			if m.mode == "help" {
				m.mode = ""
				return m, nil
			}
			if key == "enter" {
				switch m.mode {
				case "search":
					m.mode = ""
					return m, nil
				case "host":
					if e := api.ValidateHost(m.input); e != nil {
						m.operationError = e.Error()
						return m, nil
					}
					m.host = m.input
					m.operationError = ""
					m.updated = time.Time{}
					m.service, _ = transport.New(m.host)
					m.sessions = nil
					m.cursor = 0
					m.generation++
					m.mode = ""
					m.filter = ""
					m.input = ""
					m.failure = ""
					m.loading = true
					return m, m.fetch()
				case "new":
					if e := api.ValidateName(m.input); e != nil {
						m.operationError = e.Error()
						return m, nil
					}
					m.mutating = true
					m.operationError = ""
					name := m.input
					return m, func() tea.Msg { s, e := m.service.Create(m.ctx, name); return createMsg{s, e} }
				case "remove":
					s := m.pending
					if s == nil {
						return m, nil
					}
					if m.input != s.Name {
						m.operationError = "Type the session name exactly to remove it."
						return m, nil
					}
					m.mutating = true
					m.operationError = ""
					return m, func() tea.Msg { return removeMsg{m.service.Remove(m.ctx, *s), s.Name} }
				}
			}
			if key == "backspace" {
				r := []rune(m.input)
				if len(r) > 0 {
					m.input = string(r[:len(r)-1])
				}
			} else if text := v.Key().Text; text != "" && len(m.input) < 253 {
				m.input += Clean(text)
			}
			if m.mode == "search" {
				m.filter = m.input
				m.cursor = 0
			}
			return m, nil
		}
		switch key {
		case "q", "esc":
			return m, tea.Quit
		case "j", "down":
			m.cursor = min(m.cursor+1, max(0, len(m.visible())-1))
		case "k", "up":
			m.cursor = max(0, m.cursor-1)
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = max(0, len(m.visible())-1)
		case "/":
			m.mode = "search"
			m.input = m.filter
		case "n":
			m.operationError = ""
			m.mode = "new"
			m.input = ""
			m.failure = ""
		case "h":
			m.operationError = ""
			m.mode = "host"
			m.input = ""
			m.failure = ""
		case "x", "delete":
			if m.selected() != nil && m.failure == "" {
				m.mode = "remove"
				m.pending = m.selected()
				m.input = ""
				m.operationError = ""
			}
		case "r":
			if !m.loading {
				m.loading = true
				return m, m.fetch()
			}
		case "?":
			m.mode = "help"
		case "enter":
			if s := m.selected(); s != nil && m.failure == "" {
				m.choice = Choice{m.host, s}
				return m, tea.Quit
			}
		}
	}
	return m, nil
}
func Clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, ansi.Strip(s))
}
func cut(s string, n int) string { return ansi.Truncate(Clean(s), max(0, n), "…") }
func style(color string) lip.Style {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return lip.NewStyle()
	}
	return lip.NewStyle().Foreground(lip.Color(color))
}

var cyan = style("#65D9F4")
var muted = style("#8995A9")
var green = style("#82D9A5")
var red = style("#FF8D8D")
var bold = lip.NewStyle().Bold(true)

func (m Model) View() tea.View {
	w := max(20, m.width-4)
	if m.width < 44 || m.height < 16 {
		v := tea.NewView("\n  sess\n\n  Enlarge the terminal to 44 × 16.\n  Press q to quit.")
		v.AltScreen = true
		return v
	}
	var b strings.Builder
	state := "CONNECTED"
	if m.loading {
		state = []string{"◐", "◓", "◑", "◒"}[m.frame%4] + " SYNCING"
	}
	if m.failure != "" {
		state = "NEEDS ATTENTION"
	}
	b.WriteString(cyan.Bold(true).Render("sess") + muted.Render("  /  ") + bold.Render(cut(m.host, w-30)) + "  " + muted.Render(state) + "\n")
	b.WriteString(muted.Render("Persistent SSH terminals · powered by zmx") + "\n\n")
	attached := 0
	for _, s := range m.sessions {
		if s.Clients > 0 {
			attached++
		}
	}
	sessionLabel := " sessions    "
	if len(m.sessions) == 1 {
		sessionLabel = " session     "
	}
	b.WriteString(bold.Render(fmt.Sprintf("%d", len(m.sessions))) + muted.Render(sessionLabel) + green.Render(fmt.Sprintf("%d", attached)) + muted.Render(" attached    ") + cyan.Render(fmt.Sprintf("%d", len(m.sessions)-attached)) + muted.Render(" detached") + "\n")
	query := m.filter
	if query == "" {
		query = "Press / to filter sessions"
	}
	if m.mode == "search" {
		query = m.input + "▏"
	}
	b.WriteString(muted.Render("⌕ ") + cut(query, w-3) + "\n\n")
	if m.mode == "help" {
		b.WriteString(bold.Render("Make yourself at home") + "\n\n")
		b.WriteString("↑/↓ or j/k   Select a session\nEnter        Attach to your selected terminal\nn            Create and attach\nx            Remove selected session\n/            Filter by name\nh            Browse another SSH host\nr            Refresh now\nq            Quit the browser\n\n")
		b.WriteString(cyan.Render("Inside a session: Ctrl+\\ detaches your terminal.") + "\n")
		b.WriteString(muted.Render("Any key returns to your sessions."))
	} else if m.mode == "new" || m.mode == "host" || m.mode == "remove" {
		title, desc := "New session", "Start a shell in your remote home directory."
		if m.mode == "host" {
			title = "Switch host"
			desc = "SSH alias or user@host. Your saved default stays unchanged."
		}
		if m.mode == "remove" {
			title = "Remove session"
			if s := m.pending; s != nil {
				desc = "Stops its programs and disconnects all clients. Type " + s.Name + " to remove."
			}
		}
		b.WriteString(bold.Render(title) + "\n\n" + lip.NewStyle().Width(w).Render(desc) + "\n\n")
		b.WriteString(cyan.Render("› ") + cut(m.input, w-4) + "▏\n\n")
		if m.mutating {
			b.WriteString(muted.Render("Working…"))
		} else {
			b.WriteString(muted.Render("enter confirm   esc cancel"))
		}
	} else {
		nameW := max(8, min(36, w-34))
		b.WriteString(muted.Render(fmt.Sprintf("  %-*s %-10s %7s  %s", nameW, "NAME", "STATE", "CLIENTS", "AGE")) + "\n")
		b.WriteString(muted.Render(strings.Repeat("─", w)) + "\n")
		ss := m.visible()
		rows := max(1, m.height-19)
		start := max(0, m.cursor-rows+1)
		if len(ss) == 0 {
			text := "No sessions yet. Press n to start your first terminal."
			if m.loading && m.updated.IsZero() {
				text = "Connecting to your host…"
			} else if m.failure != "" {
				text = "Could not load sessions. Press r to retry or h to change host."
			} else if m.filter != "" {
				text = "No matching sessions. Press / to change your filter."
			}
			b.WriteString("\n" + lip.NewStyle().Width(w).Render(text) + "\n")
		} else {
			for i := start; i < min(len(ss), start+rows); i++ {
				s := ss[i]
				marker := "  "
				if i == m.cursor {
					marker = "› "
				}
				line := fmt.Sprintf("%s%-*s %-10s %7d  %s", marker, nameW, cut(s.Name, nameW), s.Status, s.Clients, Age(s.Created))
				if i == m.cursor {
					selectedStyle := cyan.Bold(true)
					if _, noColor := os.LookupEnv("NO_COLOR"); !noColor {
						selectedStyle = selectedStyle.Background(lip.Color("#1D3440"))
					}
					b.WriteString(selectedStyle.Width(w).Render(line))
				} else {
					b.WriteString(line)
				}
				b.WriteByte('\n')
			}
			if len(ss) > rows {
				b.WriteString(muted.Render(fmt.Sprintf("  %d–%d of %d", start+1, min(len(ss), start+rows), len(ss))) + "\n")
			}
			if s := m.selected(); s != nil {
				b.WriteByte('\n')
				b.WriteString(muted.Render("DIRECTORY  ") + cut(s.Directory, w-12) + "\n")
				b.WriteString(muted.Render("SESSION    ") + cut(s.Name+" · "+m.host, w-12) + "\n")
			}
		}
	}
	problem := m.operationError
	if problem == "" {
		problem = m.failure
	}
	if problem != "" {
		b.WriteString("\n" + red.Render(lip.NewStyle().Width(w).Render(Clean(problem))) + "\n")
	} else if m.notice != "" {
		b.WriteString("\n" + green.Render(m.notice) + "\n")
	}
	footer := "enter attach  n new  x remove  / filter  h host  ? help  q quit"
	if w < 65 {
		footer = "enter attach  n new  ? help  q quit"
	}
	gap := max(1, m.height-lip.Height(b.String())-2)
	b.WriteString(strings.Repeat("\n", gap) + muted.Render(footer))
	content := lip.NewStyle().Padding(1, 2).Render(b.String())
	// Crop defensively at tiny heights; no scrolling of the user's outer terminal.
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, m.width, "")
	}
	content = strings.Join(lines, "\n")
	if len(lines) > m.height {
		content = strings.Join(lines[:m.height], "\n")
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
func Age(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
func Run(ctx context.Context, host string, previousError string) (Choice, error) {
	c, e := transport.New(host)
	if e != nil {
		return Choice{}, e
	}
	browserCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	initial := New(browserCtx, host, c)
	initial.operationError = previousError
	model, e := tea.NewProgram(initial, tea.WithContext(browserCtx)).Run()
	if e != nil {
		return Choice{}, e
	}
	return model.(Model).choice, nil
}
