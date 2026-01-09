package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samuelenocsson/devops-tui/internal/models"
	"github.com/samuelenocsson/devops-tui/internal/ui/theme"
)

const maxVisibleOptions = 6 // Max visible options per filter group

// FilterPanel is the filter panel component
type FilterPanel struct {
	filterState *models.FilterState
	styles      theme.Styles
	keys        theme.KeyMap
	width       int
	height      int
	focused     bool
}

// NewFilterPanel creates a new filter panel
func NewFilterPanel(filterState *models.FilterState, styles theme.Styles, keys theme.KeyMap) FilterPanel {
	return FilterPanel{
		filterState: filterState,
		styles:      styles,
		keys:        keys,
	}
}

// Init initializes the filter panel
func (f FilterPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the filter panel
func (f FilterPanel) Update(msg tea.Msg) (FilterPanel, tea.Cmd) {
	if !f.focused {
		return f, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, f.keys.Up):
			if group := f.filterState.ActiveFilterGroup(); group != nil {
				group.MoveUp()
				f.adjustOffset(group)
			}
		case key.Matches(msg, f.keys.Down):
			if group := f.filterState.ActiveFilterGroup(); group != nil {
				group.MoveDown()
				f.adjustOffset(group)
			}
		case key.Matches(msg, f.keys.Top):
			if group := f.filterState.ActiveFilterGroup(); group != nil {
				group.MoveToTop()
				group.Offset = 0
			}
		case key.Matches(msg, f.keys.Bottom):
			if group := f.filterState.ActiveFilterGroup(); group != nil {
				group.MoveToBottom()
				f.adjustOffset(group)
			}
		case msg.Type == tea.KeyPgUp:
			if group := f.filterState.ActiveFilterGroup(); group != nil {
				jump := maxVisibleOptions / 2
				if jump < 1 {
					jump = 1
				}
				for i := 0; i < jump && group.Cursor > 0; i++ {
					group.MoveUp()
				}
				f.adjustOffset(group)
			}
		case msg.Type == tea.KeyPgDown:
			if group := f.filterState.ActiveFilterGroup(); group != nil {
				jump := maxVisibleOptions / 2
				if jump < 1 {
					jump = 1
				}
				for i := 0; i < jump && group.Cursor < len(group.Options)-1; i++ {
					group.MoveDown()
				}
				f.adjustOffset(group)
			}
		case key.Matches(msg, f.keys.Select):
			if group := f.filterState.ActiveFilterGroup(); group != nil {
				group.SelectCurrent()
			}
			return f, func() tea.Msg { return FilterChangedMsg{} }
		case key.Matches(msg, f.keys.Left):
			f.filterState.PrevGroup()
		case key.Matches(msg, f.keys.Right):
			f.filterState.NextGroup()
		}
	}

	return f, nil
}

// adjustOffset ensures the cursor is visible within the scroll window
func (f *FilterPanel) adjustOffset(group *models.FilterGroup) {
	if group.Cursor < group.Offset {
		group.Offset = group.Cursor
	}
	if group.Cursor >= group.Offset+maxVisibleOptions {
		group.Offset = group.Cursor - maxVisibleOptions + 1
	}
}

// View renders the filter panel
func (f FilterPanel) View() string {
	var b strings.Builder

	for i, group := range f.filterState.Groups {
		isActiveGroup := i == f.filterState.ActiveGroup && f.focused

		// Group title with count if scrollable
		titleStyle := f.styles.FilterGroupTitle
		if isActiveGroup {
			titleStyle = titleStyle.Foreground(lipgloss.Color("#7C3AED"))
		}
		title := group.Title
		if len(group.Options) > maxVisibleOptions {
			countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
			title += countStyle.Render(" (" + itoa(len(group.Options)) + ")")
		}
		b.WriteString(titleStyle.Render(title))
		b.WriteString("\n")

		// Separator
		sep := strings.Repeat("─", min(f.width-4, 15))
		b.WriteString(f.styles.Subtitle.Render(sep))
		b.WriteString("\n")

		// Scroll up indicator
		if group.Offset > 0 {
			scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
			b.WriteString(scrollStyle.Render("  ▲ more"))
			b.WriteString("\n")
		}

		// Calculate visible range
		startIdx := group.Offset
		endIdx := startIdx + maxVisibleOptions
		if endIdx > len(group.Options) {
			endIdx = len(group.Options)
		}

		// Options (only visible ones)
		for j := startIdx; j < endIdx; j++ {
			opt := group.Options[j]
			isCursor := j == group.Cursor && isActiveGroup

			// Selection indicator
			var indicator string
			if opt.Selected {
				indicator = "●"
			} else {
				indicator = "○"
			}

			// Cursor indicator
			var cursor string
			if isCursor {
				cursor = "▸"
			} else {
				cursor = " "
			}

			// Style the option
			optionStyle := f.styles.FilterOption
			if opt.Selected {
				optionStyle = f.styles.FilterSelected
			}
			if isCursor {
				optionStyle = optionStyle.Bold(true).Foreground(lipgloss.Color("#7C3AED"))
			}

			line := cursor + " " + indicator + " " + opt.Label
			b.WriteString(optionStyle.Render(line))
			b.WriteString("\n")
		}

		// Scroll down indicator
		if endIdx < len(group.Options) {
			scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
			b.WriteString(scrollStyle.Render("  ▼ more"))
			b.WriteString("\n")
		}

		// Add spacing between groups
		if i < len(f.filterState.Groups)-1 {
			b.WriteString("\n")
		}
	}

	// Apply panel styling
	content := b.String()
	if f.focused {
		return f.styles.PanelActive.
			Width(f.width).
			Height(f.height).
			Render(content)
	}
	return f.styles.PanelInactive.
		Width(f.width).
		Height(f.height).
		Render(content)
}

// SetSize sets the size of the filter panel
func (f *FilterPanel) SetSize(width, height int) {
	f.width = width
	f.height = height
}

// SetFocused sets whether the panel is focused
func (f *FilterPanel) SetFocused(focused bool) {
	f.focused = focused
}

// FilterState returns the current filter state
func (f *FilterPanel) FilterState() *models.FilterState {
	return f.filterState
}

// SetFilterState updates the filter state
func (f *FilterPanel) SetFilterState(state *models.FilterState) {
	f.filterState = state
}

// FilterChangedMsg is sent when a filter selection changes
type FilterChangedMsg struct{}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
