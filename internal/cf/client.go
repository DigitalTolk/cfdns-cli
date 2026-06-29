// Package cf is a thin wrapper around the official cloudflare-go/v4 SDK,
// exposing only the read operations this tool needs and flattening the SDK
// types into small structs the TUI can render directly.
package cf

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
)

// Client wraps a configured cloudflare-go client.
type Client struct {
	api *cloudflare.Client
}

// New builds a Client authenticated with the given API token.
func New(token string) *Client {
	return &Client{
		api: cloudflare.NewClient(option.WithAPIToken(token)),
	}
}

// Zone is a flattened view of a Cloudflare zone.
type Zone struct {
	ID     string
	Name   string
	Status string
	Paused bool
}

// Record is a flattened view of a DNS record.
type Record struct {
	ID       string
	ZoneID   string
	ZoneName string
	Type     string
	Name     string
	Content  string
	TTL      int
	Proxied  bool
	Priority int
	Comment  string
}

// Verify performs a cheap authenticated call to confirm the token works and
// has the permissions we need. It returns a friendly error otherwise.
func (c *Client) Verify(ctx context.Context) error {
	_, err := c.api.Zones.List(ctx, zones.ZoneListParams{
		PerPage: cloudflare.F(1.0),
	})
	if err != nil {
		return fmt.Errorf("token verification failed (check the token and its Zone:Read / DNS:Read permissions): %w", err)
	}
	return nil
}

// ListZones returns all zones visible to the token, paging automatically.
func (c *Client) ListZones(ctx context.Context) ([]Zone, error) {
	iter := c.api.Zones.ListAutoPaging(ctx, zones.ZoneListParams{
		PerPage: cloudflare.F(50.0),
	})

	var out []Zone
	for iter.Next() {
		z := iter.Current()
		out = append(out, Zone{
			ID:     z.ID,
			Name:   z.Name,
			Status: string(z.Status),
			Paused: z.Paused,
		})
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("listing zones: %w", err)
	}
	return out, nil
}

// ListRecords returns all DNS records for a zone, paging automatically.
func (c *Client) ListRecords(ctx context.Context, zone Zone) ([]Record, error) {
	iter := c.api.DNS.Records.ListAutoPaging(ctx, dns.RecordListParams{
		ZoneID:  cloudflare.F(zone.ID),
		PerPage: cloudflare.F(100.0),
	})

	var out []Record
	for iter.Next() {
		r := iter.Current()
		out = append(out, Record{
			ID:       r.ID,
			ZoneID:   zone.ID,
			ZoneName: zone.Name,
			Type:     string(r.Type),
			Name:     r.Name,
			Content:  r.Content,
			TTL:      int(r.TTL),
			Proxied:  r.Proxied,
			Priority: int(r.Priority),
			Comment:  r.Comment,
		})
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("listing records for %s: %w", zone.Name, err)
	}
	return out, nil
}
