package tui

import (
	"fmt"

	"github.com/DigitalTolk/cfdns-cli/internal/cf"
)

// zoneItem adapts a cf.Zone to the bubbles list.DefaultItem interface.
type zoneItem struct{ z cf.Zone }

func (i zoneItem) Title() string { return i.z.Name }

func (i zoneItem) Description() string {
	status := i.z.Status
	if status == "" {
		status = "unknown"
	}
	if i.z.Paused {
		status += ", paused"
	}
	return fmt.Sprintf("%s  -  %s", status, i.z.ID)
}

// FilterValue is what the list's fuzzy filter matches against.
func (i zoneItem) FilterValue() string { return i.z.Name }

// recordItem adapts a cf.Record to the bubbles list.DefaultItem interface.
type recordItem struct{ r cf.Record }

func (i recordItem) Title() string {
	return fmt.Sprintf("%-6s %s", i.r.Type, i.r.Name)
}

func (i recordItem) Description() string {
	desc := i.r.Content
	if i.r.Proxied {
		desc += "  -  proxied"
	}
	if i.r.Comment != "" {
		desc += "  -  " + i.r.Comment
	}
	return desc
}

// FilterValue lets the user search by type, name, or content at once.
func (i recordItem) FilterValue() string {
	return i.r.Type + " " + i.r.Name + " " + i.r.Content
}
