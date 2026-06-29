package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// The palette uses explicit hex colors (truecolor) and AdaptiveColor so the UI
// reads the same regardless of the terminal's own 16-color theme (Dracula,
// Solarized, etc.). AdaptiveColor only switches on whether the terminal
// background is light or dark — which lipgloss detects — not on the palette.
//
// Dark variants are Dracula-aligned; light variants are darker equivalents that
// stay legible on a light background.
var (
	// accent is the primary brand color (purple).
	accent = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#BD93F9"}
	// onAccent is text placed on top of an accent-colored background.
	onAccent = lipgloss.Color("#FFFFFF")
	// idColor highlights IDs — the whole point of the tool (pink).
	idColor = lipgloss.AdaptiveColor{Light: "#BE185D", Dark: "#FF79C6"}
	// match highlights filter-matched characters (green).
	match = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#50FA7B"}
	// fg is primary body text.
	fg = lipgloss.AdaptiveColor{Light: "#1F2933", Dark: "#F8F8F2"}
	// muted is secondary text (descriptions, help, labels).
	muted = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
)

// styleList overrides the list's footer (help line), status bar, and
// pagination styles, which otherwise use very dim defaults that wash out on
// some terminal themes.
func styleList(l *list.Model) {
	keyStyle := lipgloss.NewStyle().Foreground(fg)
	descStyle := lipgloss.NewStyle().Foreground(muted)

	l.Help.Styles.ShortKey = keyStyle
	l.Help.Styles.FullKey = keyStyle
	l.Help.Styles.ShortDesc = descStyle
	l.Help.Styles.FullDesc = descStyle
	l.Help.Styles.ShortSeparator = descStyle
	l.Help.Styles.FullSeparator = descStyle
	l.Help.Styles.Ellipsis = descStyle

	l.Styles.StatusBar = descStyle
	l.Styles.StatusBarFilterCount = descStyle
	l.Styles.StatusEmpty = descStyle
	l.Styles.StatusBarActiveFilter = lipgloss.NewStyle().Foreground(accent)
	l.Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(accent)
	l.Styles.InactivePaginationDot = descStyle
}

// newDelegate returns a list item delegate styled with the palette so selected,
// normal, and filter-matched rows all stay readable on any terminal theme.
func newDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	// Selected row: accent left bar + accent text.
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(accent).BorderForeground(accent)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(accent).BorderForeground(accent)

	// Unselected rows.
	d.Styles.NormalTitle = d.Styles.NormalTitle.Foreground(fg)
	d.Styles.NormalDesc = d.Styles.NormalDesc.Foreground(muted)

	// Characters that matched the current filter, on both selected and normal.
	d.Styles.FilterMatch = d.Styles.FilterMatch.Foreground(match).Bold(true)

	return d
}
