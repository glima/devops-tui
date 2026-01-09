package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samuelenocsson/devops-tui/internal/models"
	"github.com/samuelenocsson/devops-tui/internal/ui/theme"
)

// isPageUp checks if the key message is PageUp
func isPageUp(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyPgUp
}

// isPageDown checks if the key message is PageDown
func isPageDown(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyPgDown
}

// SortField represents the field to sort by
type SortField int

const (
	SortByID SortField = iota
	SortByState
	SortByType
)

// SortDirection represents the sort direction
type SortDirection int

const (
	SortAsc SortDirection = iota
	SortDesc
)

// Column definitions
type column struct {
	title    string
	width    int
	minWidth int
	flex     bool // If true, this column takes remaining space
}

// WorkItemsPanel is the work items list component
type WorkItemsPanel struct {
	items     []models.WorkItem
	cursor    int
	styles    theme.Styles
	keys      theme.KeyMap
	width     int
	height    int
	focused   bool
	offset    int // For scrolling
	columns   []column
	sortField SortField
	sortDir   SortDirection
}

// NewWorkItemsPanel creates a new work items panel
func NewWorkItemsPanel(styles theme.Styles, keys theme.KeyMap) WorkItemsPanel {
	return WorkItemsPanel{
		items:  []models.WorkItem{},
		styles: styles,
		keys:   keys,
		columns: []column{
			{title: "ID", width: 10, minWidth: 10},    // #12345678 - never truncate
			{title: "TYPE", width: 8, minWidth: 8},    // Feature, PBI, etc - never truncate
			{title: "STATE", width: 12, minWidth: 12}, // In Progress - never truncate
			{title: "ASSIGNED", width: 16, minWidth: 12},
			{title: "TITLE", flex: true, minWidth: 20},
		},
	}
}

// Init initializes the work items panel
func (w WorkItemsPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the work items panel
func (w WorkItemsPanel) Update(msg tea.Msg) (WorkItemsPanel, tea.Cmd) {
	if !w.focused {
		return w, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, w.keys.Up):
			w.moveUp()
		case key.Matches(msg, w.keys.Down):
			w.moveDown()
		case key.Matches(msg, w.keys.Top):
			w.moveToTop()
		case key.Matches(msg, w.keys.Bottom):
			w.moveToBottom()
		case isPageUp(msg):
			jump := w.visibleItemCount() / 2
			if jump < 1 {
				jump = 1
			}
			w.cursor -= jump
			if w.cursor < 0 {
				w.cursor = 0
			}
			// Adjust offset
			visible := w.visibleItemCount()
			if w.cursor < w.offset {
				w.offset = w.cursor
			}
			if w.cursor >= w.offset+visible {
				w.offset = w.cursor - visible + 1
			}
		case isPageDown(msg):
			jump := w.visibleItemCount() / 2
			if jump < 1 {
				jump = 1
			}
			w.cursor += jump
			if w.cursor >= len(w.items) {
				w.cursor = len(w.items) - 1
			}
			if w.cursor < 0 {
				w.cursor = 0
			}
			// Adjust offset
			visible := w.visibleItemCount()
			if w.cursor < w.offset {
				w.offset = w.cursor
			}
			if w.cursor >= w.offset+visible {
				w.offset = w.cursor - visible + 1
			}
		case key.Matches(msg, w.keys.Open):
			if w.SelectedItem() != nil {
				return w, func() tea.Msg { return OpenWorkItemMsg{Item: *w.SelectedItem()} }
			}
		case key.Matches(msg, w.keys.View):
			if w.SelectedItem() != nil {
				return w, func() tea.Msg { return ViewWorkItemMsg{Item: *w.SelectedItem()} }
			}
		case key.Matches(msg, w.keys.SortByID):
			w.toggleSort(SortByID)
		case key.Matches(msg, w.keys.SortByState):
			w.toggleSort(SortByState)
		case key.Matches(msg, w.keys.SortByType):
			w.toggleSort(SortByType)
		}
	}

	return w, nil
}

// View renders the work items panel
func (w WorkItemsPanel) View() string {
	var b strings.Builder

	// Calculate column widths
	colWidths := w.calculateColumnWidths()

	// Header
	header := w.renderHeader(colWidths)
	b.WriteString(header)
	b.WriteString("\n")

	// Separator line
	separator := w.renderSeparator(colWidths)
	b.WriteString(separator)
	b.WriteString("\n")

	// Items
	if len(w.items) == 0 {
		emptyMsg := w.styles.Subtitle.Render("  No work items found")
		b.WriteString(emptyMsg)
	} else {
		visibleItems := w.visibleItemCount()

		// Render visible items
		for i := w.offset; i < len(w.items) && i < w.offset+visibleItems; i++ {
			item := w.items[i]
			isCursor := i == w.cursor
			line := w.renderItem(item, isCursor, colWidths)
			b.WriteString(line)
			if i < len(w.items)-1 && i < w.offset+visibleItems-1 {
				b.WriteString("\n")
			}
		}
	}

	// Apply panel styling with MaxHeight to prevent overflow
	content := b.String()
	if w.focused {
		return w.styles.PanelActive.
			Width(w.width).
			Height(w.height).
			MaxHeight(w.height + 2).
			Render(content)
	}
	return w.styles.PanelInactive.
		Width(w.width).
		Height(w.height).
		MaxHeight(w.height + 2).
		Render(content)
}

func (w *WorkItemsPanel) calculateColumnWidths() []int {
	availableWidth := w.width - 6 // Account for borders and padding

	// Calculate fixed columns total width
	fixedWidth := 0
	flexCount := 0
	for _, col := range w.columns {
		if col.flex {
			flexCount++
		} else {
			fixedWidth += col.width + 1 // +1 for separator
		}
	}

	// Calculate flex column width
	flexWidth := availableWidth - fixedWidth
	if flexWidth < 20 {
		flexWidth = 20
	}

	// Build widths array
	widths := make([]int, len(w.columns))
	for i, col := range w.columns {
		if col.flex {
			widths[i] = flexWidth / flexCount
		} else {
			widths[i] = col.width
		}
	}

	return widths
}

func (w *WorkItemsPanel) renderHeader(colWidths []int) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#9CA3AF"))

	sortedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED"))

	var parts []string
	for i, col := range w.columns {
		width := colWidths[i]
		title := col.title

		// Add sort indicator
		isSorted := (col.title == "ID" && w.sortField == SortByID) ||
			(col.title == "STATE" && w.sortField == SortByState) ||
			(col.title == "TYPE" && w.sortField == SortByType)

		if isSorted {
			arrow := "▲"
			if w.sortDir == SortDesc {
				arrow = "▼"
			}
			title = title + arrow
		}

		if len(title) > width {
			title = title[:width]
		}

		if isSorted {
			parts = append(parts, sortedStyle.Width(width).Render(title))
		} else {
			parts = append(parts, headerStyle.Width(width).Render(title))
		}
	}

	return "  " + strings.Join(parts, "  ")
}

func (w *WorkItemsPanel) renderSeparator(colWidths []int) string {
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))

	// Use content width (accounting for panel padding)
	contentWidth := w.width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	return sepStyle.Render(strings.Repeat("─", contentWidth))
}

func (w *WorkItemsPanel) renderItem(item models.WorkItem, isCursor bool, colWidths []int) string {
	// Cursor indicator
	cursor := "  "
	if isCursor {
		cursor = "▸ "
	}

	// Format values - ID, TYPE, STATE never truncated; ASSIGNED and TITLE can be
	id := fmt.Sprintf("#%d", item.ID)
	typeStr := item.ShortType()
	stateStr := string(item.State)
	assigned := item.AssignedTo
	if assigned == "" {
		assigned = "-"
	}
	assigned = truncateStr(assigned, colWidths[3])
	title := truncateStr(item.Title, colWidths[4])

	// For cursor row, use plain text with unified background
	if isCursor {
		rowStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F9FAFB")).
			Background(lipgloss.Color("#7C3AED")). // Purple highlight
			MaxWidth(w.width - 4).
			MaxHeight(1)

		// Build plain text cells (no individual colors) - padRight for alignment
		cells := []string{
			padRight(id, colWidths[0]),
			padRight(typeStr, colWidths[1]),
			padRight(stateStr, colWidths[2]),
			padRight(assigned, colWidths[3]),
			padRight(title, colWidths[4]),
		}
		row := cursor + strings.Join(cells, "  ")
		return rowStyle.Render(row)
	}

	// Non-cursor rows with individual cell colors
	idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA"))
	typeStyle := w.styles.TypeBadge(string(item.Type))
	stateStyle := w.styles.StateBadge(string(item.State))
	assignedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB"))

	// Build cells with padRight for alignment, then apply color
	cells := []string{
		idStyle.Render(padRight(id, colWidths[0])),
		typeStyle.Render(padRight(typeStr, colWidths[1])),
		stateStyle.Render(padRight(stateStr, colWidths[2])),
		assignedStyle.Render(padRight(assigned, colWidths[3])),
		titleStyle.Render(padRight(title, colWidths[4])),
	}

	row := cursor + strings.Join(cells, "  ")
	return row
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func (w *WorkItemsPanel) visibleItemCount() int {
	visible := w.height - 5 // header, separator, borders
	if visible < 1 {
		visible = 1
	}
	return visible
}

func (w *WorkItemsPanel) moveUp() {
	if w.cursor > 0 {
		w.cursor--
		w.adjustOffset()
	}
}

func (w *WorkItemsPanel) moveDown() {
	if w.cursor < len(w.items)-1 {
		w.cursor++
		w.adjustOffset()
	}
}

func (w *WorkItemsPanel) moveToTop() {
	w.cursor = 0
	w.offset = 0
}

func (w *WorkItemsPanel) moveToBottom() {
	if len(w.items) > 0 {
		w.cursor = len(w.items) - 1
		w.adjustOffset()
	}
}

func (w *WorkItemsPanel) pageUp() {
	jump := w.visibleItemCount() / 2
	if jump < 1 {
		jump = 1
	}
	w.cursor -= jump
	if w.cursor < 0 {
		w.cursor = 0
	}
	w.adjustOffset()
}

func (w *WorkItemsPanel) pageDown() {
	jump := w.visibleItemCount() / 2
	if jump < 1 {
		jump = 1
	}
	w.cursor += jump
	if w.cursor >= len(w.items) {
		w.cursor = len(w.items) - 1
	}
	if w.cursor < 0 {
		w.cursor = 0
	}
	w.adjustOffset()
}

func (w *WorkItemsPanel) adjustOffset() {
	visible := w.visibleItemCount()
	if w.cursor < w.offset {
		w.offset = w.cursor
	}
	if w.cursor >= w.offset+visible {
		w.offset = w.cursor - visible + 1
	}
	if w.offset < 0 {
		w.offset = 0
	}
}

func (w *WorkItemsPanel) toggleSort(field SortField) {
	if w.sortField == field {
		// Toggle direction if same field
		if w.sortDir == SortAsc {
			w.sortDir = SortDesc
		} else {
			w.sortDir = SortAsc
		}
	} else {
		w.sortField = field
		w.sortDir = SortAsc
	}
	w.sortItems()
}

func (w *WorkItemsPanel) sortItems() {
	if len(w.items) == 0 {
		return
	}

	sort.SliceStable(w.items, func(i, j int) bool {
		var less bool
		switch w.sortField {
		case SortByID:
			less = w.items[i].ID < w.items[j].ID
		case SortByState:
			less = string(w.items[i].State) < string(w.items[j].State)
		case SortByType:
			less = string(w.items[i].Type) < string(w.items[j].Type)
		default:
			less = w.items[i].ID < w.items[j].ID
		}

		if w.sortDir == SortDesc {
			return !less
		}
		return less
	})
}

// SetSize sets the size of the work items panel
func (w *WorkItemsPanel) SetSize(width, height int) {
	w.width = width
	w.height = height

	// Adjust offset to keep cursor visible (now that we have correct height)
	visible := w.visibleItemCount()
	if w.cursor < w.offset {
		w.offset = w.cursor
	}
	if w.cursor >= w.offset+visible {
		w.offset = w.cursor - visible + 1
	}
	if w.offset < 0 {
		w.offset = 0
	}
}

// SetFocused sets whether the panel is focused
func (w *WorkItemsPanel) SetFocused(focused bool) {
	w.focused = focused
}

// SetItems sets the work items
func (w *WorkItemsPanel) SetItems(items []models.WorkItem) {
	// Remember currently selected item ID
	var selectedID int
	if w.cursor >= 0 && w.cursor < len(w.items) {
		selectedID = w.items[w.cursor].ID
	}

	oldLen := len(w.items)
	w.items = items

	// Re-apply current sort
	w.sortItems()

	// Only reset position if this is new data (not just a refresh)
	if oldLen == 0 && len(items) > 0 {
		w.cursor = 0
		w.offset = 0
	} else if selectedID > 0 {
		// Try to restore cursor to previously selected item
		for i, item := range w.items {
			if item.ID == selectedID {
				w.cursor = i
				break
			}
		}
	}

	// Clamp cursor to valid range
	if w.cursor >= len(items) {
		w.cursor = len(items) - 1
	}
	if w.cursor < 0 {
		w.cursor = 0
	}

	// Adjust offset to keep cursor visible
	visible := w.visibleItemCount()
	if w.cursor < w.offset {
		w.offset = w.cursor
	}
	if w.cursor >= w.offset+visible {
		w.offset = w.cursor - visible + 1
	}
}

// SelectedItem returns the currently selected work item
func (w *WorkItemsPanel) SelectedItem() *models.WorkItem {
	if w.cursor >= 0 && w.cursor < len(w.items) {
		return &w.items[w.cursor]
	}
	return nil
}

// OpenWorkItemMsg is sent when a work item should be opened in browser
type OpenWorkItemMsg struct {
	Item models.WorkItem
}

// ViewWorkItemMsg is sent when a work item should be viewed in detail
type ViewWorkItemMsg struct {
	Item models.WorkItem
}
