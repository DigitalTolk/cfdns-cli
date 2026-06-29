package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	// A filled bar sets both fg and bg explicitly, so it's readable on any theme.
	listTitleStyle = lipgloss.NewStyle().
			Background(accent).
			Foreground(onAccent).
			Bold(true).
			Padding(0, 1)

	spinnerStyle = lipgloss.NewStyle().Foreground(accent)
	statusStyle  = lipgloss.NewStyle().Foreground(match).Bold(true)
	errStyle     = lipgloss.NewStyle().Foreground(idColor).Bold(true)
	helpStyle    = lipgloss.NewStyle().Foreground(muted)

	detailTitleStyle = lipgloss.NewStyle().
				Foreground(onAccent).
				Background(accent).
				Bold(true).
				Padding(0, 1)

	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 2)

	labelStyle = lipgloss.NewStyle().Foreground(muted).Width(11)
	valueStyle = lipgloss.NewStyle().Foreground(fg)
	idStyle    = lipgloss.NewStyle().Foreground(idColor).Bold(true)
)

func (m Model) View() string {
	if m.err != nil {
		return appStyle.Render(
			errStyle.Render("Error") + "\n\n" +
				wrap(m.err.Error()) + "\n\n" +
				helpStyle.Render("r retry  -  q quit"),
		)
	}

	switch m.screen {
	case screenZones:
		if m.loading && len(m.zones.Items()) == 0 {
			return m.loadingView("Loading zones...")
		}
		return m.listView(m.zones.View())

	case screenRecords:
		if m.loading {
			return m.loadingView("Loading records for " + m.curZone.Name + "...")
		}
		return m.listView(m.records.View())

	case screenDetail:
		return m.detailView()
	}
	return ""
}

func (m Model) loadingView(label string) string {
	return appStyle.Render(m.spin.View() + " " + label)
}

// listView appends the transient status line under a list's own view.
func (m Model) listView(body string) string {
	if m.status == "" {
		return body
	}
	return body + "\n" + statusStyle.Render(m.status)
}

func (m Model) detailView() string {
	r := m.curRec

	row := func(label, val string, isID bool) string {
		v := valueStyle.Render(val)
		if isID {
			v = idStyle.Render(val)
		}
		return labelStyle.Render(label) + v
	}

	lines := []string{
		row("Record ID", r.ID, true),
		row("Zone ID", r.ZoneID, true),
		"",
		row("Zone", r.ZoneName, false),
		row("Type", r.Type, false),
		row("Name", r.Name, false),
		row("Content", r.Content, false),
		row("TTL", ttlString(r.TTL), false),
		row("Proxied", yesNo(r.Proxied), false),
	}
	if r.Type == "MX" || r.Type == "SRV" || r.Type == "URI" {
		lines = append(lines, row("Priority", strconv.Itoa(r.Priority), false))
	}
	if r.Comment != "" {
		lines = append(lines, row("Comment", r.Comment, false))
	}

	box := detailBoxStyle.Render(strings.Join(lines, "\n"))
	title := detailTitleStyle.Render(r.Type + "  " + r.Name)
	help := helpStyle.Render("c copy record ID  -  z copy zone ID  -  n copy name  -  esc back  -  q quit")

	out := title + "\n\n" + box + "\n" + help
	if m.status != "" {
		out += "\n" + statusStyle.Render(m.status)
	}
	return appStyle.Render(out)
}

func ttlString(ttl int) string {
	if ttl == 1 {
		return "auto"
	}
	return strconv.Itoa(ttl) + "s"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func wrap(s string) string {
	return lipgloss.NewStyle().Width(72).Render(s)
}
