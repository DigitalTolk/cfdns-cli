// Package tui implements the interactive terminal UI for browsing Cloudflare
// zones and DNS records, built on Bubble Tea.
package tui

import (
	"context"
	"sort"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/DigitalTolk/cfdns-cli/internal/cf"
)

type screen int

const (
	screenZones screen = iota
	screenRecords
	screenDetail
)

// Model is the root Bubble Tea model.
type Model struct {
	client *cf.Client

	screen  screen
	zones   list.Model
	records list.Model
	spin    spinner.Model

	loading bool
	err     error
	status  string // transient message, e.g. "Copied record ID"

	curZone cf.Zone
	curRec  cf.Record

	// Record type filter state for the records screen.
	allRecords []cf.Record // full set for the current zone
	recTypes   []string    // distinct types present, sorted
	typeFilter string      // "" means all types
}

// New constructs the root model. The first thing it does on Init is load zones.
// It returns the concrete Model, which satisfies tea.Model.
func New(client *cf.Client) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	typeKey := key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "cycle type"))

	return Model{
		client:  client,
		screen:  screenZones,
		zones:   newList("Zones", "enter", "records"),
		records: newList("DNS Records", "enter", "details", typeKey),
		spin:    sp,
		loading: true,
	}
}

func newList(title, openKey, openTarget string, extraKeys ...key.Binding) list.Model {
	l := list.New(nil, newDelegate(), 0, 0)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = listTitleStyle
	l.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(accent)
	l.FilterInput.Cursor.Style = lipgloss.NewStyle().Foreground(accent)
	styleList(&l)

	bindings := []key.Binding{
		key.NewBinding(key.WithKeys(openKey), key.WithHelp(openKey, "open "+openTarget)),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
	bindings = append(bindings, extraKeys...)
	l.AdditionalShortHelpKeys = func() []key.Binding { return bindings }

	return l
}

// ---- messages ----

type zonesMsg []cf.Zone

type recordsMsg struct {
	zone    cf.Zone
	records []cf.Record
}

type errMsg struct{ err error }

type copyResultMsg struct {
	label string
	err   error
}

type clearStatusMsg struct{}

// ---- commands ----

func (m Model) loadZones() tea.Cmd {
	return func() tea.Msg {
		zs, err := m.client.ListZones(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return zonesMsg(zs)
	}
}

func (m Model) loadRecords(z cf.Zone) tea.Cmd {
	return func() tea.Msg {
		rs, err := m.client.ListRecords(context.Background(), z)
		if err != nil {
			return errMsg{err}
		}
		return recordsMsg{zone: z, records: rs}
	}
}

func copyCmd(label, text string) tea.Cmd {
	return func() tea.Msg {
		return copyResultMsg{label: label, err: clipboard.WriteAll(text)}
	}
}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

// ---- bubbletea interface ----

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, m.loadZones())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h := msg.Height - 1 // reserve a line for the status/footer
		m.zones.SetSize(msg.Width, h)
		m.records.SetSize(msg.Width, h)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case zonesMsg:
		m.loading = false
		m.err = nil
		items := make([]list.Item, len(msg))
		for i, z := range msg {
			items[i] = zoneItem{z}
		}
		return m, m.zones.SetItems(items)

	case recordsMsg:
		m.loading = false
		m.err = nil
		m.curZone = msg.zone
		m.allRecords = msg.records
		m.recTypes = distinctTypes(msg.records)
		m.typeFilter = ""
		m.records.ResetSelected()
		return m, m.applyRecordItems()

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case copyResultMsg:
		if msg.err != nil {
			m.status = "Copy failed: " + msg.err.Error()
		} else {
			m.status = "Copied " + msg.label + " to clipboard"
		}
		return m, clearStatusCmd()

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Any other message (notably the list's own FilterMatchesMsg, which is what
	// actually narrows the list) must reach the active list component.
	return m.updateActiveList(msg)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// While the active list's filter input is open, let it consume every key
	// so typing a query (and esc-to-cancel) behaves normally.
	if m.activeListFiltering() {
		return m.updateActiveList(msg)
	}

	// Global keys (only when not typing a filter).
	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	switch m.screen {
	case screenZones:
		switch key {
		case "enter":
			if it, ok := m.zones.SelectedItem().(zoneItem); ok {
				m.curZone = it.z
				m.loading = true
				m.screen = screenRecords
				return m, m.loadRecords(it.z)
			}
			return m, nil
		case "r":
			m.loading = true
			return m, m.loadZones()
		}
		return m.updateActiveList(msg)

	case screenRecords:
		switch key {
		case "enter":
			if it, ok := m.records.SelectedItem().(recordItem); ok {
				m.curRec = it.r
				m.screen = screenDetail
			}
			return m, nil
		case "esc":
			// If a filter is applied, esc clears it; otherwise go back to zones.
			if m.records.FilterState() != list.Unfiltered {
				return m.updateActiveList(msg)
			}
			m.screen = screenZones
			return m, nil
		case "r":
			m.loading = true
			return m, m.loadRecords(m.curZone)
		case "t":
			m.cycleType()
			m.records.ResetSelected()
			return m, m.applyRecordItems()
		}
		return m.updateActiveList(msg)

	case screenDetail:
		switch key {
		case "esc", "enter":
			m.screen = screenRecords
			return m, nil
		case "c", "y":
			return m, copyCmd("record ID", m.curRec.ID)
		case "z":
			return m, copyCmd("zone ID", m.curRec.ZoneID)
		case "n":
			return m, copyCmd("record name", m.curRec.Name)
		}
		return m, nil
	}

	return m, nil
}

func (m Model) activeListFiltering() bool {
	switch m.screen {
	case screenZones:
		return m.zones.FilterState() == list.Filtering
	case screenRecords:
		return m.records.FilterState() == list.Filtering
	}
	return false
}

func (m Model) updateActiveList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.screen {
	case screenZones:
		m.zones, cmd = m.zones.Update(msg)
	case screenRecords:
		m.records, cmd = m.records.Update(msg)
	}
	return m, cmd
}

// applyRecordItems repopulates the records list from allRecords, honoring the
// active type filter, and reflects the filter in the list title.
func (m *Model) applyRecordItems() tea.Cmd {
	items := make([]list.Item, 0, len(m.allRecords))
	for _, r := range m.allRecords {
		if m.typeFilter == "" || r.Type == m.typeFilter {
			items = append(items, recordItem{r})
		}
	}

	title := "DNS Records - " + m.curZone.Name
	if m.typeFilter != "" {
		title += "  [type: " + m.typeFilter + "]"
	}
	m.records.Title = title

	return m.records.SetItems(items)
}

// cycleType advances the type filter through: all -> each present type -> all.
func (m *Model) cycleType() {
	options := append([]string{""}, m.recTypes...)
	idx := 0
	for i, t := range options {
		if t == m.typeFilter {
			idx = i
			break
		}
	}
	m.typeFilter = options[(idx+1)%len(options)]
}

// distinctTypes returns the sorted set of record types present.
func distinctTypes(records []cf.Record) []string {
	seen := make(map[string]struct{})
	var types []string
	for _, r := range records {
		if _, ok := seen[r.Type]; !ok {
			seen[r.Type] = struct{}{}
			types = append(types, r.Type)
		}
	}
	sort.Strings(types)
	return types
}
