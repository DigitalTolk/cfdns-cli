package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/DigitalTolk/cfdns-cli/internal/cf"
)

func TestItemRendering(t *testing.T) {
	z := zoneItem{cf.Zone{ID: "z1", Name: "example.com", Status: "active", Paused: true}}
	if z.Title() != "example.com" {
		t.Errorf("zone title = %q", z.Title())
	}
	if d := z.Description(); !strings.Contains(d, "paused") || !strings.Contains(d, "z1") {
		t.Errorf("zone desc = %q", d)
	}
	if z.FilterValue() != "example.com" {
		t.Errorf("zone filter = %q", z.FilterValue())
	}
	if d := (zoneItem{cf.Zone{Name: "x"}}).Description(); !strings.Contains(d, "unknown") {
		t.Errorf("missing status should read 'unknown', got %q", d)
	}

	r := recordItem{cf.Record{Type: "A", Name: "www", Content: "1.2.3.4", Proxied: true, Comment: "note"}}
	if ti := r.Title(); !strings.Contains(ti, "A") || !strings.Contains(ti, "www") {
		t.Errorf("record title = %q", ti)
	}
	if d := r.Description(); !strings.Contains(d, "1.2.3.4") || !strings.Contains(d, "proxied") || !strings.Contains(d, "note") {
		t.Errorf("record desc = %q", d)
	}
	if fv := r.FilterValue(); !strings.Contains(fv, "A") || !strings.Contains(fv, "www") || !strings.Contains(fv, "1.2.3.4") {
		t.Errorf("record filter = %q", fv)
	}
}

func TestFormatHelpers(t *testing.T) {
	if ttlString(1) != "auto" {
		t.Errorf("ttl 1 should be 'auto', got %q", ttlString(1))
	}
	if ttlString(3600) != "3600s" {
		t.Errorf("ttl 3600 should be '3600s', got %q", ttlString(3600))
	}
	if yesNo(true) != "yes" || yesNo(false) != "no" {
		t.Error("yesNo mapping wrong")
	}
}

func TestCopyResultStatus(t *testing.T) {
	m := recordsScreen(t)
	m.screen = screenDetail

	ok := drive(t, m, copyResultMsg{label: "record ID", err: nil})
	if !strings.Contains(ok.status, "Copied record ID") {
		t.Errorf("success status = %q", ok.status)
	}

	failed := drive(t, m, copyResultMsg{label: "zone ID", err: errExample{}})
	if !strings.Contains(failed.status, "Copy failed") {
		t.Errorf("failure status = %q", failed.status)
	}
	// The detail view should surface the transient status line.
	if !strings.Contains(failed.detailView(), "Copy failed") {
		t.Error("detail view should render the status message")
	}

	// clearStatusMsg wipes it again.
	cleared := drive(t, failed, clearStatusMsg{})
	if cleared.status != "" {
		t.Errorf("status should be cleared, got %q", cleared.status)
	}
}

func TestCopyKeysReturnCommands(t *testing.T) {
	m := recordsScreen(t)
	m = drive(t, m, keyMsg("enter")) // open detail

	for _, k := range []string{"c", "z", "n"} {
		if _, cmd := m.Update(keyMsg(k)); cmd == nil {
			t.Errorf("key %q in detail should return a copy command", k)
		}
	}
}

func TestInitAndLoadingView(t *testing.T) {
	m := New(nil)
	if m.Init() == nil {
		t.Error("Init should return a command")
	}
	sized := drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if !strings.Contains(sized.View(), "Loading zones") {
		t.Errorf("expected loading view, got:\n%s", sized.View())
	}
}

// TestLoadCommands drives loadZones/loadRecords against a fake Cloudflare API,
// covering the command closures and their message-handling arms in Update.
func TestLoadCommands(t *testing.T) {
	const empty = `{"success":true,"errors":[],"messages":[],"result":[],"result_info":{"page":2,"per_page":50,"count":0,"total_count":1,"total_pages":1}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		paged := r.URL.Query().Get("page")
		beyond := paged != "" && paged != "1"
		switch {
		case strings.HasSuffix(r.URL.Path, "/dns_records"):
			if beyond {
				w.Write([]byte(empty))
				return
			}
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":[{"id":"recA","type":"A","name":"www.example.com","content":"1.2.3.4","ttl":1}],"result_info":{"page":1,"per_page":100,"count":1,"total_count":1,"total_pages":1}}`))
		default:
			if beyond {
				w.Write([]byte(empty))
				return
			}
			w.Write([]byte(`{"success":true,"errors":[],"messages":[],"result":[{"id":"z1","name":"example.com","status":"active"}],"result_info":{"page":1,"per_page":50,"count":1,"total_count":1,"total_pages":1}}`))
		}
	}))
	defer srv.Close()
	t.Setenv("CLOUDFLARE_BASE_URL", srv.URL)

	m := New(cf.New("tok"))
	m = drive(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})

	// loadZones command -> zonesMsg -> populated zones list.
	zmsg := m.loadZones()()
	if _, ok := zmsg.(zonesMsg); !ok {
		t.Fatalf("loadZones returned %T, want zonesMsg", zmsg)
	}
	m = drive(t, m, zmsg)
	if len(m.zones.Items()) != 1 {
		t.Fatalf("expected 1 zone item, got %d", len(m.zones.Items()))
	}

	// Enter selects the zone and kicks off loadRecords; run that command too.
	next, cmd := m.Update(keyMsg("enter"))
	m = next.(Model)
	if m.screen != screenRecords || cmd == nil {
		t.Fatal("enter should switch to records and start a load")
	}
	rmsg := cmd()
	if _, ok := rmsg.(recordsMsg); !ok {
		t.Fatalf("loadRecords returned %T, want recordsMsg", rmsg)
	}
	m = drive(t, m, rmsg)
	if len(m.records.Items()) != 1 {
		t.Fatalf("expected 1 record item, got %d", len(m.records.Items()))
	}
}
