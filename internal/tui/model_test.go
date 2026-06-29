package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/DigitalTolk/cfdns-cli/internal/cf"
)

// drive applies a message and returns the concrete model, failing if the
// model type ever changes underneath us.
func drive(t *testing.T, m tea.Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	got, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned unexpected model type %T", next)
	}
	return got
}

func keyMsg(s string) tea.KeyMsg {
	if s == "enter" {
		return tea.KeyMsg{Type: tea.KeyEnter}
	}
	if s == "esc" {
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestBrowseFlow(t *testing.T) {
	m := New(nil)
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m = drive(t, m, zonesMsg{{ID: "zone123", Name: "example.com", Status: "active"}})

	if got := len(m.zones.Items()); got != 1 {
		t.Fatalf("expected 1 zone, got %d", got)
	}

	// Enter on the zone moves us to the records screen.
	m = drive(t, m, keyMsg("enter"))
	if m.screen != screenRecords {
		t.Fatalf("expected records screen, got %v", m.screen)
	}

	rec := cf.Record{ID: "rec456", ZoneID: "zone123", ZoneName: "example.com", Type: "A", Name: "www.example.com", Content: "203.0.113.10", TTL: 1, Proxied: true}
	m = drive(t, m, recordsMsg{zone: cf.Zone{ID: "zone123", Name: "example.com"}, records: []cf.Record{rec}})

	// Enter on the record opens the detail screen, which must show the IDs.
	m = drive(t, m, keyMsg("enter"))
	if m.screen != screenDetail {
		t.Fatalf("expected detail screen, got %v", m.screen)
	}
	view := m.View()
	for _, want := range []string{"rec456", "zone123", "www.example.com", "auto"} {
		if !strings.Contains(view, want) {
			t.Errorf("detail view missing %q\n%s", want, view)
		}
	}

	// esc walks back: detail -> records -> zones.
	m = drive(t, m, keyMsg("esc"))
	if m.screen != screenRecords {
		t.Fatalf("expected records screen after esc, got %v", m.screen)
	}
	m = drive(t, m, keyMsg("esc"))
	if m.screen != screenZones {
		t.Fatalf("expected zones screen after esc, got %v", m.screen)
	}
}

func TestErrorView(t *testing.T) {
	m := New(nil)
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m = drive(t, m, errMsg{err: errExample{}})
	if !strings.Contains(m.View(), "boom") {
		t.Errorf("expected error view to show the error, got:\n%s", m.View())
	}
}

type errExample struct{}

func (errExample) Error() string { return "boom" }

// collect runs a cmd (resolving tea.Batch) and returns the messages it emits.
// It must not be used with time-based cmds (tea.Tick), which would block.
func collect(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	switch v := cmd().(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range v {
			out = append(out, collect(c)...)
		}
		return out
	case nil:
		return nil
	default:
		return []tea.Msg{v}
	}
}

// drivePump applies a message and then feeds back any messages its command
// produces (one level), which is how the list delivers its FilterMatchesMsg.
func drivePump(t *testing.T, m tea.Model, msg tea.Msg) Model {
	t.Helper()
	next, cmd := m.Update(msg)
	mm := next.(Model)
	for _, rm := range collect(cmd) {
		mm = drive(t, mm, rm)
	}
	return mm
}

func sampleRecords() []cf.Record {
	return []cf.Record{
		{ID: "1", Type: "A", Name: "www.example.com", Content: "1.1.1.1"},
		{ID: "2", Type: "A", Name: "app.example.com", Content: "1.1.1.2"},
		{ID: "3", Type: "MX", Name: "example.com", Content: "mail.example.com"},
		{ID: "4", Type: "TXT", Name: "example.com", Content: "v=spf1"},
	}
}

func recordsScreen(t *testing.T) Model {
	t.Helper()
	m := New(nil)
	m = drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
	zone := cf.Zone{ID: "z", Name: "example.com"}
	m = drive(t, m, recordsMsg{zone: zone, records: sampleRecords()})
	m.screen = screenRecords
	return m
}

func TestTextFilterNarrows(t *testing.T) {
	m := recordsScreen(t)

	m = drivePump(t, m, keyMsg("/"))
	for _, r := range "txt" {
		m = drivePump(t, m, keyMsg(string(r)))
	}

	vis := m.records.VisibleItems()
	if len(vis) != 1 {
		t.Fatalf("expected 1 visible item for filter 'txt', got %d", len(vis))
	}
	if got := vis[0].(recordItem).r.Type; got != "TXT" {
		t.Fatalf("expected TXT, got %s", got)
	}
}

func TestTypeCycleFilter(t *testing.T) {
	m := recordsScreen(t)

	// recTypes sorted: [A, MX, TXT]. One press of 't' -> filter to A (2 records).
	m = drive(t, m, keyMsg("t"))
	if m.typeFilter != "A" {
		t.Fatalf("expected typeFilter A, got %q", m.typeFilter)
	}
	if got := len(m.records.Items()); got != 2 {
		t.Fatalf("expected 2 A records, got %d", got)
	}
	if !strings.Contains(m.records.Title, "[type: A]") {
		t.Errorf("title should show active type filter, got %q", m.records.Title)
	}

	// Cycle through MX, TXT, then back to all (empty).
	m = drive(t, m, keyMsg("t")) // MX
	m = drive(t, m, keyMsg("t")) // TXT
	m = drive(t, m, keyMsg("t")) // all
	if m.typeFilter != "" {
		t.Fatalf("expected cycle back to all, got %q", m.typeFilter)
	}
	if got := len(m.records.Items()); got != 4 {
		t.Fatalf("expected all 4 records, got %d", got)
	}
}
