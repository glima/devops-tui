package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/samuelenocsson/devops-tui/internal/models"
	"github.com/samuelenocsson/devops-tui/internal/ui/theme"
)

// DetailsPanel shows details for a selected work item
type DetailsPanel struct {
	item              *models.WorkItem
	styles            theme.Styles
	keys              theme.KeyMap
	width             int
	height            int
	focused           bool
	scrollOffset      int
	maxScroll         int
	renderedContent   string
	renderedDescWidth int
}

// NewDetailsPanel creates a new details panel
func NewDetailsPanel(styles theme.Styles, keys theme.KeyMap) DetailsPanel {
	return DetailsPanel{
		styles: styles,
		keys:   keys,
	}
}

// Update handles messages for the details panel
func (d DetailsPanel) Update(msg tea.Msg) (DetailsPanel, tea.Cmd) {
	if !d.focused {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.Up):
			if d.scrollOffset > 0 {
				d.scrollOffset--
			}
		case key.Matches(msg, d.keys.Down):
			if d.scrollOffset < d.maxScroll {
				d.scrollOffset++
			}
		case key.Matches(msg, d.keys.Top):
			d.scrollOffset = 0
		case key.Matches(msg, d.keys.Bottom):
			d.scrollOffset = d.maxScroll
		case msg.Type == tea.KeyPgUp:
			jump := d.height / 2
			if jump < 1 {
				jump = 1
			}
			d.scrollOffset -= jump
			if d.scrollOffset < 0 {
				d.scrollOffset = 0
			}
		case msg.Type == tea.KeyPgDown:
			jump := d.height / 2
			if jump < 1 {
				jump = 1
			}
			d.scrollOffset += jump
			if d.scrollOffset > d.maxScroll {
				d.scrollOffset = d.maxScroll
			}
		case key.Matches(msg, d.keys.Open):
			if d.item != nil {
				return d, func() tea.Msg { return OpenWorkItemMsg{Item: *d.item} }
			}
		case key.Matches(msg, d.keys.View):
			if d.item != nil {
				return d, func() tea.Msg { return ViewWorkItemMsg{Item: *d.item} }
			}
		}
	}

	return d, nil
}

// View renders the details panel
func (d DetailsPanel) View() string {
	panelStyle := d.styles.PanelInactive
	if d.focused {
		panelStyle = d.styles.PanelActive
	}

	if d.item == nil {
		content := d.styles.Subtitle.Render("Select a work item to view details")
		return panelStyle.
			Width(d.width).
			Height(d.height).
			Render(content)
	}

	// Build content
	content := d.buildContent()

	// Calculate scrolling
	contentLines := strings.Split(content, "\n")
	viewableHeight := d.height - 2 // Account for border
	d.maxScroll = len(contentLines) - viewableHeight
	if d.maxScroll < 0 {
		d.maxScroll = 0
	}

	// Apply scrolling
	if d.scrollOffset > d.maxScroll {
		d.scrollOffset = d.maxScroll
	}
	if d.scrollOffset > 0 && d.scrollOffset < len(contentLines) {
		contentLines = contentLines[d.scrollOffset:]
	}
	if len(contentLines) > viewableHeight {
		contentLines = contentLines[:viewableHeight]
	}

	scrolledContent := strings.Join(contentLines, "\n")

	// Add scroll indicator if content is scrollable
	if d.maxScroll > 0 && d.focused {
		scrollPercent := 0
		if d.maxScroll > 0 {
			scrollPercent = (d.scrollOffset * 100) / d.maxScroll
		}
		indicator := fmt.Sprintf(" [%d%%]", scrollPercent)
		scrolledContent = strings.TrimRight(scrolledContent, "\n")
		scrolledContent += "\n" + d.styles.Subtitle.Render(indicator)
	}

	return panelStyle.
		Width(d.width).
		Height(d.height).
		Render(scrolledContent)
}

func (d *DetailsPanel) buildContent() string {
	var b strings.Builder

	// Calculate dynamic widths based on panel width
	contentWidth := d.width - 6 // Account for borders and padding
	if contentWidth < 40 {
		contentWidth = 40
	}

	// Title - use full width
	title := fmt.Sprintf("#%d %s", d.item.ID, d.item.Title)
	if len(title) > contentWidth {
		title = title[:contentWidth-3] + "..."
	}
	b.WriteString(d.styles.DetailTitle.Render(title))
	b.WriteString("\n\n")

	// Metadata section - use single column layout for better readability
	typeStyle := d.styles.TypeBadge(string(d.item.Type))
	stateStyle := d.styles.StateBadge(string(d.item.State))

	labelWidth := 12
	// Value width takes remaining space
	valueWidth := contentWidth - labelWidth - 2
	if valueWidth < 20 {
		valueWidth = 20
	}

	label := func(s string) string {
		return d.styles.DetailLabel.Width(labelWidth).Render(s)
	}
	value := func(s string) string {
		return d.styles.DetailValue.Render(truncate(s, valueWidth))
	}

	// Type and State on same row (these are short)
	b.WriteString(label("Type:"))
	b.WriteString(typeStyle.Render(d.item.ShortType()))
	b.WriteString("  ")
	b.WriteString(label("State:"))
	b.WriteString(stateStyle.Render(string(d.item.State)))
	b.WriteString("  ")
	b.WriteString(label("Priority:"))
	b.WriteString(d.styles.DetailValue.Render(fmt.Sprintf("%d", d.item.Priority)))
	b.WriteString("\n")

	// Assigned - full row
	assignedTo := d.item.AssignedTo
	if assignedTo == "" {
		assignedTo = "Unassigned"
	}
	b.WriteString(label("Assigned:"))
	b.WriteString(value(assignedTo))
	b.WriteString("\n")

	// Sprint - full row
	b.WriteString(label("Sprint:"))
	b.WriteString(value(d.item.SprintName()))
	b.WriteString("\n")

	// Area - full row
	b.WriteString(label("Area:"))
	b.WriteString(value(d.item.AreaName()))
	b.WriteString("\n")

	// Dates on same row
	b.WriteString(label("Created:"))
	b.WriteString(d.styles.DetailValue.Render(d.item.CreatedDate.Format("2006-01-02")))
	b.WriteString("  ")
	b.WriteString(label("Updated:"))
	b.WriteString(d.styles.DetailValue.Render(d.item.ChangedDate.Format("2006-01-02")))
	b.WriteString("\n")

	// Created by - full row (if available)
	if d.item.CreatedBy != "" {
		b.WriteString(label("Created by:"))
		b.WriteString(value(d.item.CreatedBy))
		b.WriteString("\n")
	}

	// Changed by - full row (if available and different from created by)
	if d.item.ChangedBy != "" && d.item.ChangedBy != d.item.CreatedBy {
		b.WriteString(label("Changed by:"))
		b.WriteString(value(d.item.ChangedBy))
		b.WriteString("\n")
	}

	// Board column (if available)
	if d.item.BoardColumn != "" {
		b.WriteString(label("Board:"))
		boardValue := d.item.BoardColumn
		if d.item.BoardColumnDone {
			boardValue += " (Done)"
		}
		b.WriteString(value(boardValue))
		b.WriteString("\n")
	}

	// Estimation fields based on work item type
	if d.item.HasEstimates() {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Estimates ───"))
		b.WriteString("\n")

		switch d.item.Type {
		case models.WorkItemTypeTask:
			if d.item.OriginalEstimate > 0 {
				b.WriteString(label("Original:"))
				b.WriteString(value(fmt.Sprintf("%.1fh", d.item.OriginalEstimate)))
			}
			if d.item.CompletedWork > 0 {
				b.WriteString(label("Completed:"))
				b.WriteString(value(fmt.Sprintf("%.1fh", d.item.CompletedWork)))
			}
			if d.item.RemainingWork > 0 {
				b.WriteString(label("Remaining:"))
				b.WriteString(value(fmt.Sprintf("%.1fh", d.item.RemainingWork)))
			}
			b.WriteString("\n")
			if d.item.Activity != "" {
				b.WriteString(label("Activity:"))
				b.WriteString(value(d.item.Activity))
				b.WriteString("\n")
			}
		case models.WorkItemTypeStory, models.WorkItemTypeBug:
			if d.item.StoryPoints > 0 {
				b.WriteString(label("Story Pts:"))
				b.WriteString(value(fmt.Sprintf("%.0f", d.item.StoryPoints)))
				b.WriteString("\n")
			}
		case models.WorkItemTypeFeature, models.WorkItemTypeEpic:
			if d.item.Effort > 0 {
				b.WriteString(label("Effort:"))
				b.WriteString(value(fmt.Sprintf("%.0f", d.item.Effort)))
				b.WriteString("\n")
			}
		}
	}

	// Bug-specific fields
	if d.item.Type == models.WorkItemTypeBug && d.item.Severity != "" {
		b.WriteString(label("Severity:"))
		b.WriteString(value(d.item.Severity))
		b.WriteString("\n")
	}

	// Feature/Epic-specific fields
	if (d.item.Type == models.WorkItemTypeFeature || d.item.Type == models.WorkItemTypeEpic) && d.item.Risk != "" {
		b.WriteString(label("Risk:"))
		b.WriteString(value(d.item.Risk))
		b.WriteString("\n")
	}

	// Value Area
	if d.item.ValueArea != "" {
		b.WriteString(label("Value Area:"))
		b.WriteString(value(d.item.ValueArea))
		b.WriteString("\n")
	}

	// Reason (why current state)
	if d.item.Reason != "" {
		b.WriteString(label("Reason:"))
		b.WriteString(value(d.item.Reason))
		b.WriteString("\n")
	}

	// Parent (if exists)
	if d.item.ParentID > 0 {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Parent ───"))
		b.WriteString("\n")
		parentLabel := fmt.Sprintf("#%d", d.item.ParentID)
		if d.item.ParentTitle != "" {
			parentLabel += " " + d.item.ParentTitle
		}
		b.WriteString(d.styles.Subtitle.Render(parentLabel))
		b.WriteString("\n")
	}

	// Children (if exist)
	if len(d.item.ChildIDs) > 0 {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Children ───"))
		b.WriteString("\n")
		for _, link := range d.item.RelatedLinks {
			if link.LinkType == "Child" {
				childLabel := fmt.Sprintf("#%d", link.TargetID)
				if link.Title != "" {
					childLabel += " " + truncate(link.Title, contentWidth-15)
				}
				if link.State != "" {
					childLabel += " [" + link.State + "]"
				}
				b.WriteString(d.styles.Subtitle.Render("  • " + childLabel))
				b.WriteString("\n")
			}
		}
	}

	// Related links (excluding parent/children)
	hasOtherLinks := false
	for _, link := range d.item.RelatedLinks {
		if link.LinkType != "Child" && link.LinkType != "Parent" {
			hasOtherLinks = true
			break
		}
	}
	if hasOtherLinks {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Related ───"))
		b.WriteString("\n")
		for _, link := range d.item.RelatedLinks {
			if link.LinkType != "Child" && link.LinkType != "Parent" {
				linkLabel := fmt.Sprintf("[%s] #%d", link.LinkType, link.TargetID)
				if link.Title != "" {
					linkLabel += " " + truncate(link.Title, contentWidth-25)
				}
				b.WriteString(d.styles.Subtitle.Render("  • " + linkLabel))
				b.WriteString("\n")
			}
		}
	}

	// Description section
	if d.item.Description != "" {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Description ───"))
		b.WriteString("\n")

		desc := d.renderMarkdown(d.item.Description, contentWidth-4)
		b.WriteString(desc)
		b.WriteString("\n")
	}

	// Acceptance Criteria (for User Stories)
	if d.item.AcceptanceCriteria != "" {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Acceptance Criteria ───"))
		b.WriteString("\n")

		ac := d.renderMarkdown(d.item.AcceptanceCriteria, contentWidth-4)
		b.WriteString(ac)
		b.WriteString("\n")
	}

	// Repro Steps (for Bugs)
	if d.item.ReproSteps != "" {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Repro Steps ───"))
		b.WriteString("\n")

		repro := d.renderMarkdown(d.item.ReproSteps, contentWidth-4)
		b.WriteString(repro)
		b.WriteString("\n")
	}

	// Tags section
	if len(d.item.Tags) > 0 {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Tags ───"))
		b.WriteString("\n")

		var tagStrings []string
		for _, tag := range d.item.Tags {
			tagStrings = append(tagStrings, d.styles.DetailTag.Render(tag))
		}
		b.WriteString(strings.Join(tagStrings, " "))
		b.WriteString("\n")
	}

	// Comments section
	if len(d.item.Comments) > 0 {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render(fmt.Sprintf("─── Comments (%d) ───", len(d.item.Comments))))
		b.WriteString("\n")

		for i, comment := range d.item.Comments {
			// Comment header
			header := fmt.Sprintf("%s • %s", comment.CreatedBy, comment.CreatedDate.Format("2006-01-02 15:04"))
			b.WriteString(d.styles.DetailLabel.Render(header))
			b.WriteString("\n")

			// Comment body
			commentText := wordWrap(comment.Text, contentWidth-4)
			b.WriteString(d.styles.DetailValue.Render(commentText))
			b.WriteString("\n")

			if i < len(d.item.Comments)-1 {
				b.WriteString("\n")
			}
		}
	} else if d.item.CommentCount > 0 {
		// Show comment count even if not loaded
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render(fmt.Sprintf("─── Comments (%d) ───", d.item.CommentCount)))
		b.WriteString("\n")
		b.WriteString(d.styles.Subtitle.Render("Press 'v' to view full details with comments"))
		b.WriteString("\n")
	}

	return b.String()
}

// SetItem sets the work item to display
func (d *DetailsPanel) SetItem(item *models.WorkItem) {
	// Reset scroll when item changes
	if d.item == nil || item == nil || d.item.ID != item.ID {
		d.scrollOffset = 0
	}
	d.item = item
	d.renderedContent = ""
	d.renderedDescWidth = 0
}

// SetFocused sets the focused state of the panel
func (d *DetailsPanel) SetFocused(focused bool) {
	d.focused = focused
}

// IsFocused returns whether the panel is focused
func (d *DetailsPanel) IsFocused() bool {
	return d.focused
}

// renderMarkdown renders markdown content
func (d *DetailsPanel) renderMarkdown(content string, width int) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)

	var result string
	if err == nil {
		rendered, renderErr := renderer.Render(content)
		if renderErr == nil {
			result = strings.TrimSpace(rendered)
		} else {
			result = wordWrap(content, width)
		}
	} else {
		result = wordWrap(content, width)
	}

	return result
}

// SetSize sets the size of the details panel
func (d *DetailsPanel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Helper functions

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	currentLineLength := 0

	for i, word := range words {
		if currentLineLength+len(word)+1 > width {
			result.WriteString("\n")
			currentLineLength = 0
		} else if i > 0 {
			result.WriteString(" ")
			currentLineLength++
		}
		result.WriteString(word)
		currentLineLength += len(word)
	}

	return result.String()
}
