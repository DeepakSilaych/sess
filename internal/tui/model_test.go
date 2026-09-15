package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"errors"
	"github.com/DeepakSilaych/sess/internal/api"
	"strings"
	"testing"
	"time"
)

type fakeService struct{}

func (fakeService) List(context.Context) ([]api.Session, error)         { return nil, nil }
func (fakeService) Create(context.Context, string) (api.Session, error) { return api.Session{}, nil }
func (fakeService) Remove(context.Context, api.Session) error           { return nil }
func TestUnknownHostIsNotEmpty(t *testing.T) {
	m := New(context.Background(), "dev", fakeService{})
	u, _ := m.Update(listMsg{err: errors.New("host offline")})
	m = u.(Model)
	view := m.View().Content
	if !strings.Contains(view, "NEEDS ATTENTION") || strings.Contains(view, "No sessions yet") {
		t.Fatal(view)
	}
}
func TestStaleHostResponseIgnored(t *testing.T) {
	m := New(context.Background(), "new-host", fakeService{})
	m.generation = 2
	u, _ := m.Update(listMsg{sessions: []api.Session{{Name: "wrong"}}, generation: 1})
	if len(u.(Model).sessions) != 0 {
		t.Fatal("stale response rendered")
	}
}
func TestViews(t *testing.T) {
	m := New(context.Background(), "dev", fakeService{})
	m.loading = false
	m.sessions = []api.Session{{Name: "api", ID: "abc", Status: "detached", Directory: "/home/user/code", Created: time.Now()}}
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 32}, {Width: 60, Height: 20}, {Width: 40, Height: 12}} {
		u, _ := m.Update(size)
		view := u.(Model).View()
		if strings.Count(view.Content, "\n") >= size.Height {
			t.Fatalf("view exceeds %d rows", size.Height)
		}
	}
	m.filter = "missing"
	if !strings.Contains(m.View().Content, "No matching") {
		t.Fatal("missing filter state")
	}
}
func TestSanitizeRemoteText(t *testing.T) {
	if s := Clean("\x1b]52;c;payload\x07path\x1b[31m\n"); s != "path" {
		t.Fatalf("terminal controls leaked: %q", s)
	}
}

func TestOperationErrorSurvivesRefresh(t *testing.T) {
	m := New(context.Background(), "dev", fakeService{})
	u, _ := m.Update(createMsg{err: errors.New("name already exists")})
	m = u.(Model)
	u, _ = m.Update(listMsg{})
	m = u.(Model)
	if !strings.Contains(m.View().Content, "name already exists") {
		t.Fatal("refresh erased an operation error")
	}
}
