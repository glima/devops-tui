package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samuelenocsson/devops-tui/internal/api"
	"github.com/samuelenocsson/devops-tui/internal/config"
	"github.com/samuelenocsson/devops-tui/internal/models"
	"github.com/samuelenocsson/devops-tui/internal/ui/components"
	"github.com/samuelenocsson/devops-tui/internal/ui/theme"
	"github.com/samuelenocsson/devops-tui/pkg/browser"
	"github.com/samuelenocsson/devops-tui/pkg/git"
)

// Panel represents the active panel
type Panel int

const (
	PanelFilter Panel = iota
	PanelWorkItems
	PanelDetails
)

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewMain ViewMode = iota
	ViewDetail
)

// App is the main application model
type App struct {
	// Components
	filterPanel    components.FilterPanel
	workItemsPanel components.WorkItemsPanel
	detailsPanel   components.DetailsPanel
	detailView     *components.DetailView
	helpPanel      components.HelpPanel
	stateModal     components.StateModal
	branchModal    components.BranchModal
	assignModal    components.AssignModal

	// State
	activePanel Panel
	viewMode    ViewMode
	loading     bool
	err         error
	statusMsg   string // Temporary status message

	// Data
	iterations   []models.Iteration
	areas        []models.Area
	workItems    []models.WorkItem
	statesByType map[string][]models.WorkItemStateInfo
	teamMembers  []models.TeamMember

	// Services
	client *api.Client

	// Config
	styles theme.Styles
	keys   theme.KeyMap

	// Dimensions
	width  int
	height int
}

// NewApp creates a new application
func NewApp(client *api.Client) App {
	styles := theme.DefaultStyles()
	keys := theme.DefaultKeyMap()

	// Create empty filter state (will be populated after loading data)
	filterState := models.NewFilterState(nil, nil, nil)

	// Initialize detailView separately to get pointer
	detailView := components.NewDetailView(styles, keys)

	return App{
		filterPanel:    components.NewFilterPanel(filterState, styles, keys),
		workItemsPanel: components.NewWorkItemsPanel(styles, keys),
		detailsPanel:   components.NewDetailsPanel(styles, keys),
		detailView:     &detailView,
		helpPanel:      components.NewHelpPanel(keys, styles),
		stateModal:     components.NewStateModal(styles, keys),
		branchModal:    components.NewBranchModal(styles, keys),
		assignModal:    components.NewAssignModal(styles, keys),
		activePanel:    PanelWorkItems,
		viewMode:       ViewMain,
		loading:        true,
		client:         client,
		styles:         styles,
		keys:           keys,
	}
}

// Init initializes the application
func (a App) Init() tea.Cmd {
	return tea.Batch(
		loadDataCmd(a.client),
	)
}

// Update handles messages
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateSizes()

	case tea.KeyMsg:
		// Handle modals first (they capture all input when visible)
		if a.stateModal.IsVisible() {
			newModal, cmd := a.stateModal.Update(msg)
			a.stateModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.branchModal.IsVisible() {
			newModal, cmd := a.branchModal.Update(msg)
			a.branchModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		if a.assignModal.IsVisible() {
			newModal, cmd := a.assignModal.Update(msg)
			a.assignModal = newModal
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		// Global keys
		if key.Matches(msg, a.keys.Quit) && !a.helpPanel.IsVisible() && a.viewMode == ViewMain {
			return a, tea.Quit
		}

		if key.Matches(msg, a.keys.Help) {
			a.helpPanel.Toggle()
			return a, nil
		}

		// Close help with any key if visible
		if a.helpPanel.IsVisible() {
			if key.Matches(msg, a.keys.Back) || key.Matches(msg, a.keys.Help) {
				a.helpPanel.SetVisible(false)
			}
			return a, nil
		}

		// Handle detail view mode
		if a.viewMode == ViewDetail {
			_, cmd := a.detailView.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)
		}

		// Panel switching
		if key.Matches(msg, a.keys.NextPanel) {
			a.nextPanel()
			a.updateFocus()
		}
		if key.Matches(msg, a.keys.PrevPanel) {
			a.prevPanel()
			a.updateFocus()
		}

		// Refresh
		if key.Matches(msg, a.keys.Refresh) {
			a.loading = true
			a.statusMsg = ""
			return a, loadWorkItemsCmd(a.client, a.filterPanel.FilterState())
		}

		// Open state change modal (only when work items panel is active)
		if key.Matches(msg, a.keys.ChangeState) && a.activePanel == PanelWorkItems {
			if item := a.workItemsPanel.SelectedItem(); item != nil {
				a.stateModal.SetItem(item)
				a.stateModal.SetSize(a.width, a.height)
				a.stateModal.SetVisible(true)
				return a, nil
			}
		}

		// Open branch modal (only when work items panel is active)
		if key.Matches(msg, a.keys.CreateBranch) && a.activePanel == PanelWorkItems {
			if item := a.workItemsPanel.SelectedItem(); item != nil {
				a.branchModal.SetItem(item)
				a.branchModal.SetSize(a.width, a.height)
				a.branchModal.SetVisible(true)
				return a, nil
			}
		}

		// Open assign modal (only when work items panel is active)
		if key.Matches(msg, a.keys.Assign) && a.activePanel == PanelWorkItems {
			if item := a.workItemsPanel.SelectedItem(); item != nil {
				a.assignModal.SetItem(item)
				a.assignModal.SetMembers(a.teamMembers)
				a.assignModal.SetSize(a.width, a.height)
				a.assignModal.SetVisible(true)
				return a, nil
			}
		}

		// Update active panel
		switch a.activePanel {
		case PanelFilter:
			newFilter, cmd := a.filterPanel.Update(msg)
			a.filterPanel = newFilter
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case PanelWorkItems:
			newWorkItems, cmd := a.workItemsPanel.Update(msg)
			a.workItemsPanel = newWorkItems
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case PanelDetails:
			newDetails, cmd := a.detailsPanel.Update(msg)
			a.detailsPanel = newDetails
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case dataLoadedMsg:
		a.iterations = msg.iterations
		a.areas = msg.areas
		a.statesByType = msg.statesByType
		a.teamMembers = msg.teamMembers
		a.stateModal.SetStatesByType(a.statesByType)
		filterState := models.NewFilterState(a.iterations, a.areas, a.statesByType)

		// Apply saved filter selections
		if savedState, err := config.LoadFilterState(); err == nil {
			filterState.ApplySavedSelections(savedState.Sprint, savedState.State, savedState.Assigned, savedState.Area)
		}

		a.filterPanel.SetFilterState(filterState)
		// Load work items with initial filters
		return a, loadWorkItemsCmd(a.client, filterState)

	case workItemsLoadedMsg:
		a.loading = false
		a.workItems = msg.items
		a.workItemsPanel.SetItems(msg.items)
		a.updateSelectedItem()

	case components.FilterChangedMsg:
		a.loading = true
		fs := a.filterPanel.FilterState()

		// Save filter selections for next startup
		_ = config.SaveFilterState(&config.FilterState{
			Sprint:   fs.GetSelectedSprint(),
			State:    fs.GetSelectedState(),
			Assigned: fs.GetSelectedAssigned(),
			Area:     fs.GetSelectedArea(),
		})

		return a, loadWorkItemsCmd(a.client, fs)

	case components.OpenWorkItemMsg:
		if err := browser.Open(msg.Item.WebURL); err != nil {
			a.err = err
		}

	case components.ViewWorkItemMsg:
		a.viewMode = ViewDetail
		// Load full work item details including comments
		return a, loadFullWorkItemCmd(a.client, msg.Item.ID)

	case fullWorkItemLoadedMsg:
		a.detailView.SetItem(msg.item)
		a.updateSizes()

	case components.CloseDetailViewMsg:
		a.viewMode = ViewMain

	case errMsg:
		a.loading = false
		a.err = msg.err

	case components.ModalClosedMsg:
		// Modal was closed, nothing special to do
		a.stateModal.SetVisible(false)
		a.branchModal.SetVisible(false)
		a.assignModal.SetVisible(false)

	case components.StateChangeRequestMsg:
		a.stateModal.SetVisible(false)
		a.loading = true
		return a, updateWorkItemStateCmd(a.client, msg.Item.ID, msg.NewState, a.filterPanel.FilterState())

	case stateChangedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("State changed to %s", msg.newState)
		// Refresh work items to show updated state
		return a, loadWorkItemsCmd(a.client, a.filterPanel.FilterState())

	case components.BranchCreateRequestMsg:
		a.branchModal.SetVisible(false)
		return a, createBranchCmd(msg.BranchName)

	case components.BranchCreatedMsg:
		a.statusMsg = fmt.Sprintf("Branch created: %s", msg.BranchName)

	case components.BranchCreateErrorMsg:
		a.err = msg.Err

	case components.AssignRequestMsg:
		a.assignModal.SetVisible(false)
		a.loading = true
		return a, assignWorkItemCmd(a.client, msg.Item.ID, msg.UserEmail, msg.UserName, a.filterPanel.FilterState())

	case assignedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Assigned to %s", msg.userName)
		// Refresh work items to show updated assignment
		return a, loadWorkItemsCmd(a.client, a.filterPanel.FilterState())
	}

	// Update selected item in details panel
	a.updateSelectedItem()

	return a, tea.Batch(cmds...)
}

// View renders the application
func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	// Render state modal if visible
	if a.stateModal.IsVisible() {
		return a.stateModal.View()
	}

	// Render branch modal if visible
	if a.branchModal.IsVisible() {
		return a.branchModal.View()
	}

	// Render assign modal if visible
	if a.assignModal.IsVisible() {
		return a.assignModal.View()
	}

	// Render help overlay if visible
	if a.helpPanel.IsVisible() {
		_ = a.renderMainView()
		help := a.helpPanel.View()
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, help)
	}

	// Render detail view if in detail mode
	if a.viewMode == ViewDetail {
		return a.detailView.View()
	}

	return a.renderMainView()
}

func (a *App) renderMainView() string {
	// Calculate dimensions
	// Borders add 2 chars per panel (1 left + 1 right)
	// Two panels side by side = 4 total border overhead
	filterWidth := int(float64(a.width) * 0.20)
	if filterWidth < 20 {
		filterWidth = 20
	}
	contentWidth := a.width - filterWidth - 4

	// Available height: total - title bar (1) - status bar (1) = a.height - 2
	// Filter panel: content height + border (2) = available height
	availableHeight := a.height - 2
	filterContentHeight := availableHeight - 2

	// Right side has two panels stacked, each with border (2 each = 4 total)
	rightContentHeight := availableHeight - 4
	workItemsHeight := int(float64(rightContentHeight) * 0.55)
	if workItemsHeight < 8 {
		workItemsHeight = 8
	}
	detailsHeight := rightContentHeight - workItemsHeight

	// Title bar
	title := a.styles.Title.Render("devops-tui")
	projectInfo := a.styles.Subtitle.Render(fmt.Sprintf("%s/%s", a.client.Organization(), a.client.Project()))
	titleBar := lipgloss.JoinHorizontal(lipgloss.Left, title, "  ", projectInfo)

	// Loading indicator
	if a.loading {
		titleBar += "  " + a.styles.Subtitle.Render("Loading...")
	}

	// Error display
	if a.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
		titleBar += "  " + errStyle.Render(a.err.Error())
	}

	// Status message
	if a.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
		titleBar += "  " + statusStyle.Render(a.statusMsg)
	}

	// Set panel sizes (content dimensions, borders added by styles)
	a.filterPanel.SetSize(filterWidth, filterContentHeight)
	a.workItemsPanel.SetSize(contentWidth, workItemsHeight)
	a.detailsPanel.SetSize(contentWidth, detailsHeight)

	// Render panels
	filterView := a.filterPanel.View()
	workItemsView := a.workItemsPanel.View()
	detailsView := a.detailsPanel.View()

	// Right side (work items + details)
	rightSide := lipgloss.JoinVertical(lipgloss.Left, workItemsView, detailsView)

	// Main content
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, filterView, rightSide)

	// Status bar
	statusBar := a.renderStatusBar()

	// Combine all
	return lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		mainContent,
		statusBar,
	)
}

func (a *App) renderStatusBar() string {
	var parts []string

	// Panel indicator
	panelName := "Filter"
	switch a.activePanel {
	case PanelWorkItems:
		panelName = "Work Items"
	case PanelDetails:
		panelName = "Details"
	}
	parts = append(parts, a.styles.HelpKey.Render("Panel")+": "+panelName)

	// Short help
	help := components.ShortHelp(a.keys, a.styles)
	parts = append(parts, help)

	return a.styles.StatusBar.Width(a.width).Render(strings.Join(parts, "  "))
}

func (a *App) nextPanel() {
	switch a.activePanel {
	case PanelFilter:
		a.activePanel = PanelWorkItems
	case PanelWorkItems:
		a.activePanel = PanelDetails
	case PanelDetails:
		a.activePanel = PanelFilter
	}
}

func (a *App) prevPanel() {
	switch a.activePanel {
	case PanelFilter:
		a.activePanel = PanelDetails
	case PanelWorkItems:
		a.activePanel = PanelFilter
	case PanelDetails:
		a.activePanel = PanelWorkItems
	}
}

func (a *App) updateFocus() {
	a.filterPanel.SetFocused(a.activePanel == PanelFilter)
	a.workItemsPanel.SetFocused(a.activePanel == PanelWorkItems)
	a.detailsPanel.SetFocused(a.activePanel == PanelDetails)
}

func (a *App) updateSizes() {
	a.helpPanel.SetSize(a.width, a.height)
	a.detailView.SetSize(a.width, a.height)
	a.updateFocus()
}

func (a *App) updateSelectedItem() {
	item := a.workItemsPanel.SelectedItem()
	a.detailsPanel.SetItem(item)
}

// Message types

type dataLoadedMsg struct {
	iterations   []models.Iteration
	areas        []models.Area
	statesByType map[string][]models.WorkItemStateInfo
	teamMembers  []models.TeamMember
}

type workItemsLoadedMsg struct {
	items []models.WorkItem
}

type fullWorkItemLoadedMsg struct {
	item *models.WorkItem
}

type errMsg struct {
	err error
}

type stateChangedMsg struct {
	newState string
}

type assignedMsg struct {
	userName string
}

// Commands

func loadDataCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		iterations, err := client.GetIterations()
		if err != nil {
			return errMsg{err: err}
		}
		areas, err := client.GetAreas()
		if err != nil {
			return errMsg{err: err}
		}
		statesByType, err := client.GetAllWorkItemTypeStates()
		if err != nil {
			// Non-fatal - we can still work with hardcoded states
			statesByType = make(map[string][]models.WorkItemStateInfo)
		}
		teamMembers, err := client.GetTeamMembers()
		if err != nil {
			// Non-fatal - we can still work without team members
			teamMembers = []models.TeamMember{}
		}
		return dataLoadedMsg{iterations: iterations, areas: areas, statesByType: statesByType, teamMembers: teamMembers}
	}
}

func loadWorkItemsCmd(client *api.Client, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		sprint := filterState.GetSelectedSprint()
		state := filterState.GetSelectedState()
		assigned := filterState.GetSelectedAssigned()
		area := filterState.GetSelectedArea()

		items, err := client.QueryWorkItems(sprint, state, assigned, area)
		if err != nil {
			return errMsg{err: err}
		}
		return workItemsLoadedMsg{items: items}
	}
}

func loadFullWorkItemCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		item, err := client.GetWorkItem(id)
		if err != nil {
			return errMsg{err: err}
		}
		return fullWorkItemLoadedMsg{item: item}
	}
}

func updateWorkItemStateCmd(client *api.Client, itemID int, newState string, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateWorkItemState(itemID, newState)
		if err != nil {
			return errMsg{err: err}
		}
		return stateChangedMsg{newState: newState}
	}
}

func createBranchCmd(branchName string) tea.Cmd {
	return func() tea.Msg {
		if !git.IsGitRepo() {
			return components.BranchCreateErrorMsg{Err: fmt.Errorf("not a git repository")}
		}
		if git.HasUncommittedChanges() {
			return components.BranchCreateErrorMsg{Err: fmt.Errorf("uncommitted changes exist")}
		}
		err := git.CreateBranch(branchName, true)
		if err != nil {
			return components.BranchCreateErrorMsg{Err: err}
		}
		return components.BranchCreatedMsg{BranchName: branchName}
	}
}

func assignWorkItemCmd(client *api.Client, itemID int, userEmail, userName string, filterState *models.FilterState) tea.Cmd {
	return func() tea.Msg {
		err := client.AssignWorkItem(itemID, userEmail)
		if err != nil {
			return errMsg{err: err}
		}
		return assignedMsg{userName: userName}
	}
}
