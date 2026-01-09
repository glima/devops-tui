package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/samuelenocsson/devops-tui/internal/models"
)

// wiqlRequest represents a WIQL query request
type wiqlRequest struct {
	Query string `json:"query"`
}

// wiqlResponse represents the response from a WIQL query
type wiqlResponse struct {
	WorkItems []struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	} `json:"workItems"`
}

// workItemsResponse represents the response for batch work item fetch
type workItemsResponse struct {
	Count int               `json:"count"`
	Value []workItemAPIItem `json:"value"`
}

// workItemAPIItem represents a work item from the API
type workItemAPIItem struct {
	ID        int                `json:"id"`
	Rev       int                `json:"rev"`
	Fields    workItemFields     `json:"fields"`
	URL       string             `json:"url"`
	Relations []workItemRelation `json:"relations,omitempty"`
}

type workItemRelation struct {
	Rel        string                 `json:"rel"`
	URL        string                 `json:"url"`
	Attributes map[string]interface{} `json:"attributes"`
}

type workItemFields struct {
	ID           int    `json:"System.Id"`
	Title        string `json:"System.Title"`
	State        string `json:"System.State"`
	Reason       string `json:"System.Reason"`
	WorkItemType string `json:"System.WorkItemType"`
	AssignedTo   *struct {
		DisplayName string `json:"displayName"`
		UniqueName  string `json:"uniqueName"`
	} `json:"System.AssignedTo"`
	CreatedBy *struct {
		DisplayName string `json:"displayName"`
		UniqueName  string `json:"uniqueName"`
	} `json:"System.CreatedBy"`
	ChangedBy *struct {
		DisplayName string `json:"displayName"`
		UniqueName  string `json:"uniqueName"`
	} `json:"System.ChangedBy"`
	IterationPath string    `json:"System.IterationPath"`
	AreaPath      string    `json:"System.AreaPath"`
	Description   string    `json:"System.Description"`
	Tags          string    `json:"System.Tags"`
	Parent        int       `json:"System.Parent"`
	Priority      int       `json:"Microsoft.VSTS.Common.Priority"`
	CreatedDate   time.Time `json:"System.CreatedDate"`
	ChangedDate   time.Time `json:"System.ChangedDate"`
	CommentCount  int       `json:"System.CommentCount"`

	// Additional fields for different work item types
	AcceptanceCriteria string  `json:"Microsoft.VSTS.Common.AcceptanceCriteria"`
	ReproSteps         string  `json:"Microsoft.VSTS.TCM.ReproSteps"`
	StoryPoints        float64 `json:"Microsoft.VSTS.Scheduling.StoryPoints"`
	Effort             float64 `json:"Microsoft.VSTS.Scheduling.Effort"`
	RemainingWork      float64 `json:"Microsoft.VSTS.Scheduling.RemainingWork"`
	CompletedWork      float64 `json:"Microsoft.VSTS.Scheduling.CompletedWork"`
	OriginalEstimate   float64 `json:"Microsoft.VSTS.Scheduling.OriginalEstimate"`
	Activity           string  `json:"Microsoft.VSTS.Common.Activity"`
	Severity           string  `json:"Microsoft.VSTS.Common.Severity"`
	ValueArea          string  `json:"Microsoft.VSTS.Common.ValueArea"`
	Risk               string  `json:"Microsoft.VSTS.Common.Risk"`
	BoardColumn        string  `json:"System.BoardColumn"`
	BoardColumnDone    bool    `json:"System.BoardColumnDone"`
}

// commentsResponse represents the response from the comments API
type commentsResponse struct {
	TotalCount int              `json:"totalCount"`
	Count      int              `json:"count"`
	Comments   []commentAPIItem `json:"comments"`
}

type commentAPIItem struct {
	ID         int    `json:"id"`
	WorkItemID int    `json:"workItemId"`
	Text       string `json:"text"`
	CreatedBy  *struct {
		DisplayName string `json:"displayName"`
		UniqueName  string `json:"uniqueName"`
	} `json:"createdBy"`
	CreatedDate time.Time `json:"createdDate"`
	ModifiedBy  *struct {
		DisplayName string `json:"displayName"`
		UniqueName  string `json:"uniqueName"`
	} `json:"modifiedBy"`
	ModifiedDate time.Time `json:"modifiedDate"`
}

// escapeWIQL escapes a string value for use in WIQL queries
func escapeWIQL(s string) string {
	// Escape single quotes by doubling them
	return strings.ReplaceAll(s, "'", "''")
}

// QueryWorkItems queries work items using WIQL
func (c *Client) QueryWorkItems(sprintPath, state, assigned, areaPath string) ([]models.WorkItem, error) {
	// Build WIQL query
	query := `SELECT [System.Id], [System.Title], [System.State], [System.WorkItemType]
FROM WorkItems
WHERE [System.TeamProject] = @project`

	// Add sprint filter
	if sprintPath != "" && sprintPath != "all" {
		query += fmt.Sprintf(`
  AND [System.IterationPath] = '%s'`, escapeWIQL(sprintPath))
	}

	// Add state filter
	if state != "" && state != "all" {
		query += fmt.Sprintf(`
  AND [System.State] = '%s'`, escapeWIQL(state))
	}

	// Add assigned filter
	if assigned == "me" {
		query += `
  AND [System.AssignedTo] = @me`
	}

	// Add area filter
	if areaPath != "" && areaPath != "all" {
		// Clean up the path
		areaPath = strings.TrimPrefix(areaPath, "\\")
		areaPath = strings.TrimSuffix(areaPath, "\\")
		query += fmt.Sprintf(`
  AND [System.AreaPath] UNDER '%s'`, escapeWIQL(areaPath))
	}

	query += `
ORDER BY [System.ChangedDate] DESC`

	// Execute WIQL query
	reqBody := wiqlRequest{Query: query}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling WIQL request: %w", err)
	}

	resp, err := c.post("/wit/wiql", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	var wiqlResp wiqlResponse
	if err := decode(resp, &wiqlResp); err != nil {
		return nil, err
	}

	if len(wiqlResp.WorkItems) == 0 {
		return []models.WorkItem{}, nil
	}

	// Get the IDs
	ids := make([]string, 0, len(wiqlResp.WorkItems))
	for _, wi := range wiqlResp.WorkItems {
		ids = append(ids, fmt.Sprintf("%d", wi.ID))
	}

	// Fetch the full work items
	return c.GetWorkItems(ids)
}

// allWorkItemFields returns all fields we want to fetch
func allWorkItemFields() string {
	return strings.Join([]string{
		"System.Id",
		"System.Title",
		"System.State",
		"System.Reason",
		"System.WorkItemType",
		"System.AssignedTo",
		"System.CreatedBy",
		"System.ChangedBy",
		"System.IterationPath",
		"System.AreaPath",
		"System.Description",
		"System.Tags",
		"System.Parent",
		"System.CommentCount",
		"System.BoardColumn",
		"System.BoardColumnDone",
		"System.CreatedDate",
		"System.ChangedDate",
		"Microsoft.VSTS.Common.Priority",
		"Microsoft.VSTS.Common.AcceptanceCriteria",
		"Microsoft.VSTS.TCM.ReproSteps",
		"Microsoft.VSTS.Scheduling.StoryPoints",
		"Microsoft.VSTS.Scheduling.Effort",
		"Microsoft.VSTS.Scheduling.RemainingWork",
		"Microsoft.VSTS.Scheduling.CompletedWork",
		"Microsoft.VSTS.Scheduling.OriginalEstimate",
		"Microsoft.VSTS.Common.Activity",
		"Microsoft.VSTS.Common.Severity",
		"Microsoft.VSTS.Common.ValueArea",
		"Microsoft.VSTS.Common.Risk",
	}, ",")
}

// GetWorkItems fetches multiple work items by ID
func (c *Client) GetWorkItems(ids []string) ([]models.WorkItem, error) {
	if len(ids) == 0 {
		return []models.WorkItem{}, nil
	}

	// API has a limit of 200 items per request
	const batchSize = 200
	var allItems []models.WorkItem

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		batch := ids[i:end]
		fields := allWorkItemFields()

		// Note: Can't use $expand=relations with fields parameter
		endpoint := fmt.Sprintf("/wit/workitems?ids=%s&fields=%s", strings.Join(batch, ","), fields)
		resp, err := c.get(endpoint)
		if err != nil {
			return nil, err
		}

		var apiResp workItemsResponse
		if err := decode(resp, &apiResp); err != nil {
			return nil, err
		}

		for _, item := range apiResp.Value {
			wi := c.convertWorkItem(item)
			allItems = append(allItems, wi)
		}
	}

	// Fetch parent titles
	c.populateParentTitles(allItems)

	return allItems, nil
}

// populateParentTitles fetches titles for all parent work items
func (c *Client) populateParentTitles(items []models.WorkItem) {
	// Collect unique parent IDs
	parentIDs := make(map[int]bool)
	for _, item := range items {
		if item.ParentID > 0 {
			parentIDs[item.ParentID] = true
		}
	}

	if len(parentIDs) == 0 {
		return
	}

	// Convert to string slice
	ids := make([]string, 0, len(parentIDs))
	for id := range parentIDs {
		ids = append(ids, fmt.Sprintf("%d", id))
	}

	// Fetch parent work items (only need ID and Title)
	endpoint := fmt.Sprintf("/wit/workitems?ids=%s&fields=System.Id,System.Title", strings.Join(ids, ","))
	resp, err := c.get(endpoint)
	if err != nil {
		return // Silently fail - parent titles are optional
	}

	var apiResp workItemsResponse
	if err := decode(resp, &apiResp); err != nil {
		return
	}

	// Build ID -> Title map
	titleMap := make(map[int]string)
	for _, item := range apiResp.Value {
		titleMap[item.ID] = item.Fields.Title
	}

	// Update items with parent titles
	for i := range items {
		if items[i].ParentID > 0 {
			if title, ok := titleMap[items[i].ParentID]; ok {
				items[i].ParentTitle = title
			}
		}
	}
}

// GetWorkItem fetches a single work item by ID with full details
func (c *Client) GetWorkItem(id int) (*models.WorkItem, error) {
	// Use $expand=all to get relations - can't combine with fields parameter
	endpoint := fmt.Sprintf("/wit/workitems/%d?$expand=all", id)
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, err
	}

	var item workItemAPIItem
	if err := decode(resp, &item); err != nil {
		return nil, err
	}

	wi := c.convertWorkItem(item)

	// Fetch parent title if parent exists
	if wi.ParentID > 0 {
		parentEndpoint := fmt.Sprintf("/wit/workitems/%d?fields=System.Title", wi.ParentID)
		parentResp, err := c.get(parentEndpoint)
		if err == nil {
			var parentItem workItemAPIItem
			if decode(parentResp, &parentItem) == nil {
				wi.ParentTitle = parentItem.Fields.Title
			}
		}
	}

	// Always fetch comments - CommentCount may not be reliable with $expand=all
	comments, err := c.GetWorkItemComments(id)
	if err == nil {
		wi.Comments = comments
		// Update comment count from actual fetched comments
		if len(comments) > 0 {
			wi.CommentCount = len(comments)
		}
	}

	// Populate related links with details
	c.populateRelatedLinks(&wi)

	return &wi, nil
}

// GetWorkItemComments fetches comments for a work item
func (c *Client) GetWorkItemComments(id int) ([]models.Comment, error) {
	endpoint := fmt.Sprintf("/wit/workitems/%d/comments", id)
	resp, err := c.getPreview(endpoint)
	if err != nil {
		return nil, err
	}

	var apiResp commentsResponse
	if err := decode(resp, &apiResp); err != nil {
		return nil, err
	}

	comments := make([]models.Comment, 0, len(apiResp.Comments))
	for _, c := range apiResp.Comments {
		comment := models.Comment{
			ID:           c.ID,
			Text:         stripHTML(c.Text),
			CreatedDate:  c.CreatedDate,
			ModifiedDate: c.ModifiedDate,
		}
		if c.CreatedBy != nil {
			comment.CreatedBy = c.CreatedBy.DisplayName
		}
		if c.ModifiedBy != nil {
			comment.ModifiedBy = c.ModifiedBy.DisplayName
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

// populateRelatedLinks fetches details for related work items
func (c *Client) populateRelatedLinks(item *models.WorkItem) {
	if len(item.RelatedLinks) == 0 {
		return
	}

	// Collect IDs that need detail fetching
	ids := make([]string, 0, len(item.RelatedLinks))
	for _, link := range item.RelatedLinks {
		if link.TargetID > 0 {
			ids = append(ids, fmt.Sprintf("%d", link.TargetID))
		}
	}

	if len(ids) == 0 {
		return
	}

	// Fetch related work items
	endpoint := fmt.Sprintf("/wit/workitems?ids=%s&fields=System.Id,System.Title,System.State,System.WorkItemType", strings.Join(ids, ","))
	resp, err := c.get(endpoint)
	if err != nil {
		return
	}

	var apiResp workItemsResponse
	if err := decode(resp, &apiResp); err != nil {
		return
	}

	// Build ID -> details map
	detailMap := make(map[int]workItemAPIItem)
	for _, wi := range apiResp.Value {
		detailMap[wi.ID] = wi
	}

	// Update related links with details
	for i := range item.RelatedLinks {
		if detail, ok := detailMap[item.RelatedLinks[i].TargetID]; ok {
			item.RelatedLinks[i].Title = detail.Fields.Title
			item.RelatedLinks[i].State = detail.Fields.State
			item.RelatedLinks[i].Type = detail.Fields.WorkItemType
		}
	}
}

// convertWorkItem converts an API work item to our model
func (c *Client) convertWorkItem(item workItemAPIItem) models.WorkItem {
	wi := models.WorkItem{
		ID:            item.ID,
		Rev:           item.Rev,
		Title:         item.Fields.Title,
		State:         models.WorkItemState(item.Fields.State),
		Type:          models.WorkItemType(item.Fields.WorkItemType),
		IterationPath: item.Fields.IterationPath,
		AreaPath:      item.Fields.AreaPath,
		Description:   stripHTML(item.Fields.Description),
		ParentID:      item.Fields.Parent,
		Priority:      item.Fields.Priority,
		CreatedDate:   item.Fields.CreatedDate,
		ChangedDate:   item.Fields.ChangedDate,
		URL:           item.URL,
		WebURL:        c.WorkItemWebURL(item.ID),
		Reason:        item.Fields.Reason,
		CommentCount:  item.Fields.CommentCount,

		// Additional fields
		AcceptanceCriteria: stripHTML(item.Fields.AcceptanceCriteria),
		ReproSteps:         stripHTML(item.Fields.ReproSteps),
		StoryPoints:        item.Fields.StoryPoints,
		Effort:             item.Fields.Effort,
		RemainingWork:      item.Fields.RemainingWork,
		CompletedWork:      item.Fields.CompletedWork,
		OriginalEstimate:   item.Fields.OriginalEstimate,
		Activity:           item.Fields.Activity,
		Severity:           item.Fields.Severity,
		ValueArea:          item.Fields.ValueArea,
		Risk:               item.Fields.Risk,
		BoardColumn:        item.Fields.BoardColumn,
		BoardColumnDone:    item.Fields.BoardColumnDone,
	}

	if item.Fields.AssignedTo != nil {
		wi.AssignedTo = item.Fields.AssignedTo.DisplayName
		wi.AssignedEmail = item.Fields.AssignedTo.UniqueName
	}

	if item.Fields.CreatedBy != nil {
		wi.CreatedBy = item.Fields.CreatedBy.DisplayName
	}

	if item.Fields.ChangedBy != nil {
		wi.ChangedBy = item.Fields.ChangedBy.DisplayName
	}

	// Parse tags
	if item.Fields.Tags != "" {
		tags := strings.Split(item.Fields.Tags, ";")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				wi.Tags = append(wi.Tags, tag)
			}
		}
	}

	// Parse relations
	if len(item.Relations) > 0 {
		for _, rel := range item.Relations {
			link := c.parseRelation(rel)
			if link != nil {
				if link.LinkType == "Child" {
					wi.ChildIDs = append(wi.ChildIDs, link.TargetID)
				}
				wi.RelatedLinks = append(wi.RelatedLinks, *link)
			}
		}
	}

	return wi
}

// parseRelation parses a work item relation into a RelatedLink
func (c *Client) parseRelation(rel workItemRelation) *models.RelatedLink {
	// Get link type name
	linkType := getLinkTypeName(rel.Rel)
	if linkType == "" {
		return nil // Unknown or unsupported link type
	}

	// Extract target ID from URL
	// URL format: https://dev.azure.com/{org}/{project}/_apis/wit/workItems/{id}
	targetID := extractWorkItemID(rel.URL)
	if targetID == 0 {
		return nil
	}

	return &models.RelatedLink{
		LinkType: linkType,
		TargetID: targetID,
		URL:      rel.URL,
	}
}

// getLinkTypeName converts a relation type to a human-readable name
func getLinkTypeName(rel string) string {
	switch rel {
	case "System.LinkTypes.Hierarchy-Forward":
		return "Child"
	case "System.LinkTypes.Hierarchy-Reverse":
		return "Parent"
	case "System.LinkTypes.Related":
		return "Related"
	case "System.LinkTypes.Dependency-Forward":
		return "Successor"
	case "System.LinkTypes.Dependency-Reverse":
		return "Predecessor"
	case "Microsoft.VSTS.Common.Affects-Forward":
		return "Affects"
	case "Microsoft.VSTS.Common.Affects-Reverse":
		return "Affected By"
	case "System.LinkTypes.Duplicate-Forward":
		return "Duplicate"
	case "System.LinkTypes.Duplicate-Reverse":
		return "Duplicate Of"
	default:
		// Skip artifact links (like pull requests, commits, etc.)
		if strings.HasPrefix(rel, "ArtifactLink") {
			return ""
		}
		return rel
	}
}

// extractWorkItemID extracts the work item ID from an API URL
func extractWorkItemID(url string) int {
	// URL format: https://dev.azure.com/{org}/{project}/_apis/wit/workItems/{id}
	parts := strings.Split(url, "/")
	if len(parts) < 1 {
		return 0
	}
	lastPart := parts[len(parts)-1]
	var id int
	fmt.Sscanf(lastPart, "%d", &id)
	return id
}

// UpdateWorkItemState updates a work item's state
func (c *Client) UpdateWorkItemState(id int, newState string) error {
	// Azure DevOps uses JSON Patch format
	patchDoc := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/fields/System.State",
			"value": newState,
		},
	}

	bodyBytes, err := json.Marshal(patchDoc)
	if err != nil {
		return fmt.Errorf("marshaling patch document: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workitems/%d", id)
	resp, err := c.patch(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// AssignWorkItem assigns a work item to a user
// Pass empty string to unassign
func (c *Client) AssignWorkItem(id int, userEmail string) error {
	// Azure DevOps uses JSON Patch format
	patchDoc := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/fields/System.AssignedTo",
			"value": userEmail,
		},
	}

	bodyBytes, err := json.Marshal(patchDoc)
	if err != nil {
		return fmt.Errorf("marshaling patch document: %w", err)
	}

	endpoint := fmt.Sprintf("/wit/workitems/%d", id)
	resp, err := c.patch(endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// stripHTML removes HTML tags from a string
func stripHTML(s string) string {
	// Convert block-level HTML tags to newlines before stripping
	// Handle <br>, <br/>, <br />
	brRe := regexp.MustCompile(`(?i)<br\s*/?>`)
	s = brRe.ReplaceAllString(s, "\n")

	// Handle </p>, </div>, </li> - add newline after closing tags
	blockCloseRe := regexp.MustCompile(`(?i)</(?:p|div|li|tr)>`)
	s = blockCloseRe.ReplaceAllString(s, "\n")

	// Handle </h1> through </h6> - add double newline
	headerCloseRe := regexp.MustCompile(`(?i)</h[1-6]>`)
	s = headerCloseRe.ReplaceAllString(s, "\n\n")

	// Now remove all remaining HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")

	// Replace common HTML entities
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")

	// Decode numeric HTML entities (like &#128230; for emoji)
	numericRe := regexp.MustCompile(`&#(\d+);`)
	s = numericRe.ReplaceAllStringFunc(s, func(match string) string {
		var num int
		fmt.Sscanf(match, "&#%d;", &num)
		if num > 0 && num <= 0x10FFFF {
			return string(rune(num))
		}
		return match
	})

	// Collapse multiple consecutive newlines into at most two
	multiNewlineRe := regexp.MustCompile(`\n{3,}`)
	s = multiNewlineRe.ReplaceAllString(s, "\n\n")

	// Trim whitespace from each line but preserve newlines
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	s = strings.Join(lines, "\n")

	// Trim leading/trailing whitespace from the whole string
	s = strings.TrimSpace(s)

	return s
}
