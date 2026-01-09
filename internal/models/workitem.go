package models

import (
	"fmt"
	"time"
)

// WorkItemType represents the type of work item
type WorkItemType string

const (
	WorkItemTypeStory   WorkItemType = "User Story"
	WorkItemTypeTask    WorkItemType = "Task"
	WorkItemTypeBug     WorkItemType = "Bug"
	WorkItemTypeFeature WorkItemType = "Feature"
	WorkItemTypeEpic    WorkItemType = "Epic"
)

// WorkItemState represents the state of a work item
type WorkItemState string

const (
	WorkItemStateNew      WorkItemState = "New"
	WorkItemStateActive   WorkItemState = "Active"
	WorkItemStateResolved WorkItemState = "Resolved"
	WorkItemStateClosed   WorkItemState = "Closed"
)

// Comment represents a work item comment
type Comment struct {
	ID           int       `json:"id"`
	Text         string    `json:"text"`
	CreatedBy    string    `json:"createdBy"`
	CreatedDate  time.Time `json:"createdDate"`
	ModifiedBy   string    `json:"modifiedBy"`
	ModifiedDate time.Time `json:"modifiedDate"`
}

// RelatedLink represents a link to another work item or resource
type RelatedLink struct {
	LinkType string `json:"linkType"` // e.g., "Parent", "Child", "Related", "Predecessor", "Successor"
	TargetID int    `json:"targetId"`
	Title    string `json:"title"`
	State    string `json:"state"`
	Type     string `json:"type"`
	URL      string `json:"url"`
}

// WorkItem represents an Azure DevOps work item
type WorkItem struct {
	ID            int           `json:"id"`
	Rev           int           `json:"rev"`
	Title         string        `json:"title"`
	State         WorkItemState `json:"state"`
	Type          WorkItemType  `json:"type"`
	AssignedTo    string        `json:"assignedTo"`
	AssignedEmail string        `json:"assignedEmail"`
	IterationPath string        `json:"iterationPath"`
	AreaPath      string        `json:"areaPath"`
	Description   string        `json:"description"`
	Tags          []string      `json:"tags"`
	ParentID      int           `json:"parentId"`
	ParentTitle   string        `json:"parentTitle"`
	Priority      int           `json:"priority"`
	CreatedDate   time.Time     `json:"createdDate"`
	ChangedDate   time.Time     `json:"changedDate"`
	CreatedBy     string        `json:"createdBy"`
	ChangedBy     string        `json:"changedBy"`
	URL           string        `json:"url"`
	WebURL        string        `json:"webUrl"`

	// Additional fields from Azure DevOps
	AcceptanceCriteria string  `json:"acceptanceCriteria"` // User Story acceptance criteria
	ReproSteps         string  `json:"reproSteps"`         // Bug reproduction steps
	StoryPoints        float64 `json:"storyPoints"`        // Story points (User Story, Bug)
	Effort             float64 `json:"effort"`             // Effort (Feature, Epic)
	RemainingWork      float64 `json:"remainingWork"`      // Remaining work hours (Task)
	CompletedWork      float64 `json:"completedWork"`      // Completed work hours (Task)
	OriginalEstimate   float64 `json:"originalEstimate"`   // Original estimate hours (Task)
	Activity           string  `json:"activity"`           // Activity type (Task)
	Severity           string  `json:"severity"`           // Severity (Bug)
	ValueArea          string  `json:"valueArea"`          // Business value area
	Risk               string  `json:"risk"`               // Risk level (Feature)
	Reason             string  `json:"reason"`             // State change reason
	BoardColumn        string  `json:"boardColumn"`        // Current board column
	BoardColumnDone    bool    `json:"boardColumnDone"`    // Is in done sub-column
	CommentCount       int     `json:"commentCount"`       // Number of comments

	// Relations
	Comments     []Comment     `json:"comments"`
	RelatedLinks []RelatedLink `json:"relatedLinks"`
	ChildIDs     []int         `json:"childIds"`
}

// ShortType returns a short version of the work item type
func (w *WorkItem) ShortType() string {
	switch w.Type {
	case WorkItemTypeStory:
		return "Story"
	case WorkItemTypeTask:
		return "Task"
	case WorkItemTypeBug:
		return "Bug"
	case WorkItemTypeFeature:
		return "Feature"
	case WorkItemTypeEpic:
		return "Epic"
	case "Product Backlog Item":
		return "PBI"
	default:
		return string(w.Type)
	}
}

// SprintName extracts the sprint name from the iteration path
func (w *WorkItem) SprintName() string {
	// IterationPath is like "MyProject\\Sprint 42"
	// Return just "Sprint 42"
	for i := len(w.IterationPath) - 1; i >= 0; i-- {
		if w.IterationPath[i] == '\\' {
			return w.IterationPath[i+1:]
		}
	}
	return w.IterationPath
}

// AreaName extracts the area name from the area path
func (w *WorkItem) AreaName() string {
	for i := len(w.AreaPath) - 1; i >= 0; i-- {
		if w.AreaPath[i] == '\\' {
			return w.AreaPath[i+1:]
		}
	}
	return w.AreaPath
}

// HasEstimates returns true if this work item type has estimation fields
func (w *WorkItem) HasEstimates() bool {
	switch w.Type {
	case WorkItemTypeTask:
		return w.RemainingWork > 0 || w.CompletedWork > 0 || w.OriginalEstimate > 0
	case WorkItemTypeStory, WorkItemTypeBug:
		return w.StoryPoints > 0
	case WorkItemTypeFeature, WorkItemTypeEpic:
		return w.Effort > 0
	}
	return false
}

// GetEstimateLabel returns the appropriate estimate label for this work item type
func (w *WorkItem) GetEstimateLabel() string {
	switch w.Type {
	case WorkItemTypeTask:
		return "Work"
	case WorkItemTypeStory, WorkItemTypeBug:
		return "Story Points"
	case WorkItemTypeFeature, WorkItemTypeEpic:
		return "Effort"
	}
	return "Estimate"
}

// GetEstimateValue returns the primary estimate value formatted as a string
func (w *WorkItem) GetEstimateValue() string {
	switch w.Type {
	case WorkItemTypeTask:
		if w.OriginalEstimate > 0 {
			return formatFloat(w.CompletedWork) + "/" + formatFloat(w.OriginalEstimate) + "h"
		}
		if w.RemainingWork > 0 {
			return formatFloat(w.RemainingWork) + "h remaining"
		}
	case WorkItemTypeStory, WorkItemTypeBug:
		if w.StoryPoints > 0 {
			return formatFloat(w.StoryPoints)
		}
	case WorkItemTypeFeature, WorkItemTypeEpic:
		if w.Effort > 0 {
			return formatFloat(w.Effort)
		}
	}
	return ""
}

// formatFloat formats a float nicely (removes trailing zeros)
func formatFloat(f float64) string {
	if f == float64(int(f)) {
		return fmt.Sprintf("%d", int(f))
	}
	return fmt.Sprintf("%.1f", f)
}

// WorkItemStateInfo represents state metadata from Azure DevOps
type WorkItemStateInfo struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	Category string `json:"category"` // Proposed, InProgress, Resolved, Completed, Removed
}
