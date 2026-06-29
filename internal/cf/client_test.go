package cf

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient spins up a fake Cloudflare API and returns a Client pointed at
// it. The SDK reads CLOUDFLARE_BASE_URL when no explicit base URL is passed.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CLOUDFLARE_BASE_URL", srv.URL)
	return New("test-token")
}

const emptyPageBody = `{"success": true, "errors": [], "messages": [], "result": [],
  "result_info": {"page": 2, "per_page": 100, "count": 0, "total_count": 2, "total_pages": 1}}`

// beyondFirstPage reports whether the auto-pager is asking for a page past the
// first; the fake API must return an empty page there or paging never ends.
func beyondFirstPage(r *http.Request) bool {
	page := r.URL.Query().Get("page")
	return page != "" && page != "1"
}

const zonesBody = `{
  "success": true, "errors": [], "messages": [],
  "result": [
    {"id": "zone1", "name": "example.com", "status": "active", "paused": false},
    {"id": "zone2", "name": "paused.test", "status": "active", "paused": true}
  ],
  "result_info": {"page": 1, "per_page": 50, "count": 2, "total_count": 2, "total_pages": 1}
}`

const recordsBody = `{
  "success": true, "errors": [], "messages": [],
  "result": [
    {"id": "recA", "type": "A", "name": "www.example.com", "content": "203.0.113.10", "ttl": 1, "proxied": true},
    {"id": "recMX", "type": "MX", "name": "example.com", "content": "mail.example.com", "ttl": 3600, "priority": 10, "comment": "primary"}
  ],
  "result_info": {"page": 1, "per_page": 100, "count": 2, "total_count": 2, "total_pages": 1}
}`

func TestListZones(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zones" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("missing/incorrect auth header: %q", got)
		}
		if beyondFirstPage(r) {
			w.Write([]byte(emptyPageBody))
			return
		}
		w.Write([]byte(zonesBody))
	})

	zones, err := c.ListZones(context.Background())
	if err != nil {
		t.Fatalf("ListZones: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
	}
	if zones[0].ID != "zone1" || zones[0].Name != "example.com" || zones[0].Status != "active" {
		t.Errorf("zone0 flattened wrong: %+v", zones[0])
	}
	if !zones[1].Paused {
		t.Errorf("zone1 should be paused: %+v", zones[1])
	}
}

func TestListRecords(t *testing.T) {
	zone := Zone{ID: "zone1", Name: "example.com"}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/zones/zone1/dns_records"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		if beyondFirstPage(r) {
			w.Write([]byte(emptyPageBody))
			return
		}
		w.Write([]byte(recordsBody))
	})

	recs, err := c.ListRecords(context.Background(), zone)
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}

	a := recs[0]
	if a.ID != "recA" || a.Type != "A" || a.Content != "203.0.113.10" {
		t.Errorf("A record flattened wrong: %+v", a)
	}
	if a.TTL != 1 || !a.Proxied {
		t.Errorf("A record ttl/proxied wrong: %+v", a)
	}
	// Zone metadata is threaded onto each record for the detail view.
	if a.ZoneID != "zone1" || a.ZoneName != "example.com" {
		t.Errorf("zone metadata not propagated: %+v", a)
	}

	mx := recs[1]
	if mx.Type != "MX" || mx.Priority != 10 || mx.TTL != 3600 || mx.Comment != "primary" {
		t.Errorf("MX record flattened wrong: %+v", mx)
	}
}

func TestVerifyOK(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(zonesBody))
	})
	if err := c.Verify(context.Background()); err != nil {
		t.Fatalf("Verify should succeed, got %v", err)
	}
}

func TestVerifyFailsOnUnauthorized(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"success": false, "errors": [{"code": 9109, "message": "Unauthorized"}]}`))
	})
	err := c.Verify(context.Background())
	if err == nil {
		t.Fatal("expected Verify to fail on 403")
	}
	if !strings.Contains(err.Error(), "token verification failed") {
		t.Errorf("expected a friendly verification error, got %v", err)
	}
}

func TestListZonesPropagatesError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success": false, "errors": [{"code": 0, "message": "boom"}]}`))
	})
	if _, err := c.ListZones(context.Background()); err == nil {
		t.Fatal("expected ListZones to return an error on 500")
	}
}
