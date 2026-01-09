package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/samuelenocsson/devops-tui/internal/models"
	"github.com/samuelenocsson/devops-tui/internal/ui/theme"
)

// DetailView is the fullscreen detail view component
type DetailView struct {
	item         *models.WorkItem
	styles       theme.Styles
	keys         theme.KeyMap
	width        int
	height       int
	scrollOffset int
	maxScroll    int
	contentLines []string
	contentBuilt bool
}

// NewDetailView creates a new detail view
func NewDetailView(styles theme.Styles, keys theme.KeyMap) DetailView {
	return DetailView{
		styles: styles,
		keys:   keys,
	}
}

// Init initializes the detail view
func (d DetailView) Init() tea.Cmd {
	return nil
}

// Update handles messages for the detail view
func (d *DetailView) Update(msg tea.Msg) (*DetailView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.Back):
			return d, func() tea.Msg { return CloseDetailViewMsg{} }
		case key.Matches(msg, d.keys.Quit) && msg.String() == "q":
			return d, func() tea.Msg { return CloseDetailViewMsg{} }
		case key.Matches(msg, d.keys.Open):
			if d.item != nil {
				return d, func() tea.Msg { return OpenWorkItemMsg{Item: *d.item} }
			}
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
			// Page up - jump by half the viewable height
			jump := (d.height - 4) / 2
			if jump < 1 {
				jump = 1
			}
			d.scrollOffset -= jump
			if d.scrollOffset < 0 {
				d.scrollOffset = 0
			}
		case msg.Type == tea.KeyPgDown:
			// Page down - jump by half the viewable height
			jump := (d.height - 4) / 2
			if jump < 1 {
				jump = 1
			}
			d.scrollOffset += jump
			if d.scrollOffset > d.maxScroll {
				d.scrollOffset = d.maxScroll
			}
		}
	}

	return d, nil
}

// View renders the detail view
func (d *DetailView) View() string {
	if d.item == nil {
		return ""
	}

	// Build content if not already built
	if !d.contentBuilt {
		d.buildContent()
	}

	// Calculate viewable area
	viewableHeight := d.height - 4
	if viewableHeight < 1 {
		viewableHeight = 1
	}

	// Calculate max scroll
	d.maxScroll = len(d.contentLines) - viewableHeight
	if d.maxScroll < 0 {
		d.maxScroll = 0
	}

	// Clamp scroll offset
	if d.scrollOffset > d.maxScroll {
		d.scrollOffset = d.maxScroll
	}
	if d.scrollOffset < 0 {
		d.scrollOffset = 0
	}

	// Get visible lines
	startLine := d.scrollOffset
	endLine := startLine + viewableHeight
	if endLine > len(d.contentLines) {
		endLine = len(d.contentLines)
	}

	visibleLines := d.contentLines[startLine:endLine]
	scrolledContent := strings.Join(visibleLines, "\n")

	// Status bar
	statusBar := d.renderStatusBar()

	// Build final view
	mainContent := d.styles.PanelActive.
		Width(d.width).
		Height(d.height - 2).
		Render(scrolledContent)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		statusBar,
	)
}

func (d *DetailView) buildContent() {
	var sections []string

	// Title bar
	title := fmt.Sprintf("#%d %s", d.item.ID, d.item.Title)
	titleBar := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F9FAFB")).
		Background(lipgloss.Color("#7C3AED")).
		Padding(0, 1).
		Width(d.width - 2).
		Render(title)
	sections = append(sections, titleBar)

	// Metadata section
	metadataContent := d.renderMetadata()
	metadataSection := d.styles.DetailSection.
		Width(d.width - 6).
		Render("METADATA\n" + metadataContent)
	sections = append(sections, metadataSection)

	// Estimates section (if applicable)
	if d.item.HasEstimates() {
		estimatesContent := d.renderEstimates()
		if estimatesContent != "" {
			estimatesSection := d.styles.DetailSection.
				Width(d.width - 6).
				Render("ESTIMATES\n" + estimatesContent)
			sections = append(sections, estimatesSection)
		}
	}

	// Parent section (if exists)
	if d.item.ParentID > 0 {
		parentContent := fmt.Sprintf("#%d", d.item.ParentID)
		if d.item.ParentTitle != "" {
			parentContent += " " + d.item.ParentTitle
		}
		parentSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("PARENT\n" + parentContent)
		sections = append(sections, parentSection)
	}

	// Children section (if exist)
	if len(d.item.ChildIDs) > 0 {
		childrenContent := d.renderChildren()
		childrenSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render(fmt.Sprintf("CHILDREN (%d)\n%s", len(d.item.ChildIDs), childrenContent))
		sections = append(sections, childrenSection)
	}

	// Related links section (excluding parent/children)
	relatedContent := d.renderRelatedLinks()
	if relatedContent != "" {
		relatedSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("RELATED ITEMS\n" + relatedContent)
		sections = append(sections, relatedSection)
	}

	// Description section
	if d.item.Description != "" {
		desc := d.renderMarkdown(d.item.Description, d.width-10)
		descSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("DESCRIPTION\n" + desc)
		sections = append(sections, descSection)
	}

	// Acceptance Criteria section (for User Stories)
	if d.item.AcceptanceCriteria != "" {
		ac := d.renderMarkdown(d.item.AcceptanceCriteria, d.width-10)
		acSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("ACCEPTANCE CRITERIA\n" + ac)
		sections = append(sections, acSection)
	}

	// Repro Steps section (for Bugs)
	if d.item.ReproSteps != "" {
		repro := d.renderMarkdown(d.item.ReproSteps, d.width-10)
		reproSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("REPRO STEPS\n" + repro)
		sections = append(sections, reproSection)
	}

	// Tags section
	if len(d.item.Tags) > 0 {
		var tagStrings []string
		for _, tag := range d.item.Tags {
			tagStrings = append(tagStrings, d.styles.DetailTag.Render(tag))
		}
		tagsSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render("TAGS\n" + strings.Join(tagStrings, " "))
		sections = append(sections, tagsSection)
	}

	// Comments section
	if len(d.item.Comments) > 0 {
		commentsContent := d.renderComments()
		commentsSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render(fmt.Sprintf("COMMENTS (%d)\n%s", len(d.item.Comments), commentsContent))
		sections = append(sections, commentsSection)
	} else if d.item.CommentCount > 0 {
		// Show count but comments not loaded
		commentsSection := d.styles.DetailSection.
			Width(d.width - 6).
			Render(fmt.Sprintf("COMMENTS (%d)\n%s", d.item.CommentCount, d.styles.Subtitle.Render("Comments available but not loaded")))
		sections = append(sections, commentsSection)
	}

	// Join all sections and split into lines
	content := strings.Join(sections, "\n\n")
	d.contentLines = strings.Split(content, "\n")
	d.contentBuilt = true
}

func (d *DetailView) renderMetadata() string {
	typeStyle := d.styles.TypeBadge(string(d.item.Type))
	stateStyle := d.styles.StateBadge(string(d.item.State))

	// Compact layout with fixed-width columns for proper alignment
	labelStyle := d.styles.DetailLabel
	valueStyle := d.styles.DetailValue

	// Column widths: label width + value width for each of 3 columns
	const (
		labelW1 = 12 // First column label width
		valueW1 = 22 // First column value width
		labelW2 = 12 // Second column label width
		valueW2 = 22 // Second column value width
		labelW3 = 10 // Third column label width
	)

	// Helper to create a field with padded label and value
	field := func(lbl string, val string, labelWidth, valueWidth int) string {
		paddedLabel := padRight(lbl, labelWidth)
		paddedValue := padRight(val, valueWidth)
		return labelStyle.Render(paddedLabel) + valueStyle.Render(paddedValue)
	}

	// Helper for styled values (Type, State)
	fieldStyled := func(lbl string, val string, labelWidth, valueWidth int, style lipgloss.Style) string {
		paddedLabel := padRight(lbl, labelWidth)
		paddedValue := padRight(val, valueWidth)
		return labelStyle.Render(paddedLabel) + style.Render(paddedValue)
	}

	sep := " │ "
	var rows []string

	// Row 1: Type, State, Priority
	row1 := fieldStyled("Type:", d.item.ShortType(), labelW1, valueW1, typeStyle) + sep +
		fieldStyled("State:", string(d.item.State), labelW2, valueW2, stateStyle) + sep +
		field("Priority:", fmt.Sprintf("%d", d.item.Priority), labelW3, 0)
	rows = append(rows, row1)

	// Row 2: Assigned, Area, Sprint
	assignedTo := d.item.AssignedTo
	if assignedTo == "" {
		assignedTo = "Unassigned"
	}
	row2 := field("Assigned:", assignedTo, labelW1, valueW1) + sep +
		field("Area:", d.item.AreaName(), labelW2, valueW2) + sep +
		field("Sprint:", d.item.SprintName(), labelW3, 0)
	rows = append(rows, row2)

	// Row 3: Created, Updated, Board (if available)
	boardValue := ""
	if d.item.BoardColumn != "" {
		boardValue = d.item.BoardColumn
		if d.item.BoardColumnDone {
			boardValue += " (Done)"
		}
	}
	row3 := field("Created:", d.item.CreatedDate.Format("2006-01-02"), labelW1, valueW1) + sep +
		field("Updated:", d.item.ChangedDate.Format("2006-01-02"), labelW2, valueW2)
	if boardValue != "" {
		row3 += sep + field("Board:", boardValue, labelW3, 0)
	}
	rows = append(rows, row3)

	// Row 4: Created by, Changed by (if available)
	if d.item.CreatedBy != "" || d.item.ChangedBy != "" {
		row4 := field("Created by:", d.item.CreatedBy, labelW1, valueW1) + sep +
			field("Changed by:", d.item.ChangedBy, labelW2, 0)
		rows = append(rows, row4)
	}

	// Row 5: Type-specific fields (Reason, Severity, Risk, Value Area)
	var extraFields []string
	if d.item.Reason != "" {
		extraFields = append(extraFields, field("Reason:", d.item.Reason, labelW1, valueW1))
	}
	if d.item.Type == models.WorkItemTypeBug && d.item.Severity != "" {
		extraFields = append(extraFields, field("Severity:", d.item.Severity, labelW2, valueW2))
	}
	if (d.item.Type == models.WorkItemTypeFeature || d.item.Type == models.WorkItemTypeEpic) && d.item.Risk != "" {
		extraFields = append(extraFields, field("Risk:", d.item.Risk, labelW2, valueW2))
	}
	if d.item.ValueArea != "" {
		extraFields = append(extraFields, field("Value Area:", d.item.ValueArea, labelW2, 0))
	}
	if len(extraFields) > 0 {
		rows = append(rows, strings.Join(extraFields, sep))
	}

	return strings.Join(rows, "\n")
}

func (d *DetailView) renderEstimates() string {
	labelWidth := 14
	valueWidth := 12

	label := func(s string) string {
		return d.styles.DetailLabel.Width(labelWidth).Render(s)
	}
	value := func(s string) string {
		return d.styles.DetailValue.Width(valueWidth).Render(s)
	}

	var rows []string

	switch d.item.Type {
	case models.WorkItemTypeTask:
		if d.item.OriginalEstimate > 0 {
			rows = append(rows, label("Original:")+value(fmt.Sprintf("%.1f hours", d.item.OriginalEstimate)))
		}
		if d.item.CompletedWork > 0 {
			rows = append(rows, label("Completed:")+value(fmt.Sprintf("%.1f hours", d.item.CompletedWork)))
		}
		if d.item.RemainingWork > 0 {
			rows = append(rows, label("Remaining:")+value(fmt.Sprintf("%.1f hours", d.item.RemainingWork)))
		}
		if d.item.Activity != "" {
			rows = append(rows, label("Activity:")+value(d.item.Activity))
		}
	case models.WorkItemTypeStory, models.WorkItemTypeBug:
		if d.item.StoryPoints > 0 {
			rows = append(rows, label("Story Points:")+value(fmt.Sprintf("%.0f", d.item.StoryPoints)))
		}
	case models.WorkItemTypeFeature, models.WorkItemTypeEpic:
		if d.item.Effort > 0 {
			rows = append(rows, label("Effort:")+value(fmt.Sprintf("%.0f", d.item.Effort)))
		}
	}

	return strings.Join(rows, "\n")
}

func (d *DetailView) renderChildren() string {
	var lines []string
	for _, link := range d.item.RelatedLinks {
		if link.LinkType == "Child" {
			childLine := fmt.Sprintf("  #%d", link.TargetID)
			if link.Title != "" {
				childLine += " " + link.Title
			}
			if link.State != "" {
				childLine += " [" + link.State + "]"
			}
			if link.Type != "" {
				childLine += " (" + link.Type + ")"
			}
			lines = append(lines, childLine)
		}
	}
	return strings.Join(lines, "\n")
}

func (d *DetailView) renderRelatedLinks() string {
	var lines []string
	for _, link := range d.item.RelatedLinks {
		if link.LinkType != "Child" && link.LinkType != "Parent" {
			linkLine := fmt.Sprintf("  [%s] #%d", link.LinkType, link.TargetID)
			if link.Title != "" {
				linkLine += " " + link.Title
			}
			if link.State != "" {
				linkLine += " [" + link.State + "]"
			}
			lines = append(lines, linkLine)
		}
	}
	return strings.Join(lines, "\n")
}

func (d *DetailView) renderComments() string {
	var lines []string
	maxWidth := d.width - 12
	if maxWidth < 40 {
		maxWidth = 40
	}

	for i, comment := range d.item.Comments {
		// Comment header with author and date
		header := fmt.Sprintf("  %s • %s", comment.CreatedBy, comment.CreatedDate.Format("2006-01-02 15:04"))
		lines = append(lines, d.styles.DetailLabel.Render(header))

		// Comment body - preserve original line breaks, then wrap long lines
		// Replace \r\n and \r with \n, then split
		commentText := strings.ReplaceAll(comment.Text, "\r\n", "\n")
		commentText = strings.ReplaceAll(commentText, "\r", "\n")
		paragraphs := strings.Split(commentText, "\n")
		for _, para := range paragraphs {
			if para == "" {
				lines = append(lines, "")
				continue
			}
			// Wrap each paragraph individually
			wrappedPara := wordWrap(para, maxWidth)
			for _, line := range strings.Split(wrappedPara, "\n") {
				lines = append(lines, "    "+line)
			}
		}

		// Add separator between comments
		if i < len(d.item.Comments)-1 {
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

func (d *DetailView) renderMarkdown(content string, width int) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)

	if err == nil {
		rendered, renderErr := renderer.Render(content)
		if renderErr == nil {
			return strings.TrimSpace(rendered)
		}
	}

	return wordWrap(content, width)
}

func (d *DetailView) renderStatusBar() string {
	scrollInfo := ""
	if d.maxScroll > 0 {
		scrollPercent := 0
		if d.maxScroll > 0 {
			scrollPercent = (d.scrollOffset * 100) / d.maxScroll
		}
		scrollInfo = fmt.Sprintf("  [%d%%]", scrollPercent)
	}
	help := "Esc Back  Enter Open in browser  j/k Scroll  g/G Top/Bottom  PgUp/PgDn" + scrollInfo
	return d.styles.StatusBar.
		Width(d.width).
		Render(help)
}

// SetItem sets the work item to display
func (d *DetailView) SetItem(item *models.WorkItem) {
	d.item = item
	d.scrollOffset = 0
	d.maxScroll = 0
	d.contentBuilt = false
	d.contentLines = nil
}

// SetSize sets the size of the detail view
func (d *DetailView) SetSize(width, height int) {
	// Invalidate content if size changed
	if d.width != width || d.height != height {
		d.contentBuilt = false
	}
	d.width = width
	d.height = height
}

// CloseDetailViewMsg is sent when the detail view should be closed
type CloseDetailViewMsg struct{}
