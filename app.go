package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type SourceMode string

const (
	SourceStdin       SourceMode = "stdin"
	SourceAppInsights SourceMode = "app-insights"
)

type SourceState struct {
	Mode               SourceMode     `json:"mode"`
	AppInsightsEnabled bool           `json:"appInsightsEnabled"`
	SelectedResource   *AzureResource `json:"selectedResource,omitempty"`
	Query              string         `json:"query"`
	Total              int            `json:"total"`
}

type LoadingState struct {
	Active  bool   `json:"active"`
	Message string `json:"message,omitempty"`
}

type ServerSnapshot struct {
	Source         SourceState
	TimeRange      TimeRange
	Fields         []string
	Lines          []LogLine
	Total          int
	Loading        LoadingState
	AzureResources []AzureResource
	AzureError     string
}

type App struct {
	mu                 sync.RWMutex
	buffer             *RingBuffer
	hub                *Hub
	azure              *AzureClient
	appInsightsEnabled bool
	sourceMode         SourceMode
	queryText          string
	queryAST           QueryNode
	timeRange          TimeRange
	selectedResource   *AzureResource
	azureResources     []AzureResource
	azureError         string
	fields             []string
	results            []LogLine
	total              int
	loading            LoadingState
}

func NewApp(maxLines int, appInsightsEnabled bool) *App {
	app := &App{
		buffer:             NewRingBuffer(maxLines),
		azure:              NewAzureClient(),
		appInsightsEnabled: appInsightsEnabled,
		sourceMode:         SourceStdin,
		timeRange:          DefaultTimeRange(),
	}
	if appInsightsEnabled {
		app.sourceMode = SourceAppInsights
	}
	app.hub = NewHub(app.HandleCommand)
	app.fields = app.buffer.Fields()
	return app
}

func (a *App) ServeWS(w http.ResponseWriter, r *http.Request) {
	a.hub.ServeWS(w, r, a.Snapshot())
}

func (a *App) Snapshot() ServerSnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return ServerSnapshot{
		Source:         a.sourceStateLocked(),
		TimeRange:      a.timeRange,
		Fields:         append([]string(nil), a.fields...),
		Lines:          cloneLines(a.results),
		Total:          a.total,
		Loading:        a.loading,
		AzureResources: append([]AzureResource(nil), a.azureResources...),
		AzureError:     a.azureError,
	}
}

func (a *App) HandleCommand(_ *Client, command clientCommand) {
	switch command.Type {
	case "query.apply":
		go a.ApplyQuery(command.Query)
	case "timeRange.set":
		if command.TimeRange == nil {
			a.hub.BroadcastQueryError(QueryError{Message: "Missing time range", Position: 0})
			return
		}
		go a.SetTimeRange(*command.TimeRange)
	case "source.select":
		go a.SelectSource(SourceMode(command.Source))
	case "azure.resources.list":
		go a.ListAzureResources()
	case "azure.resource.select":
		go a.SelectAzureResource(command.ResourceID)
	}
}

func (a *App) AddStdin(fields map[string]any, raw bool) {
	line, fieldsChanged, allFields := a.buffer.Add(fields, raw, time.Now())

	a.mu.Lock()
	if fieldsChanged {
		a.fields = allFields
	}
	if a.sourceMode != SourceStdin {
		a.mu.Unlock()
		return
	}

	a.total = a.buffer.Len()
	total := a.total
	timeRange := a.timeRange
	query := a.queryAST
	a.mu.Unlock()

	if fieldsChanged {
		a.hub.BroadcastFields(allFields)
	}
	a.hub.BroadcastResultsStats(total)

	matches, err := lineMatches(line, query, timeRange, time.Now())
	if err != nil {
		a.hub.BroadcastQueryError(QueryError{Message: err.Error(), Position: 0})
		return
	}
	if !matches {
		return
	}

	a.mu.Lock()
	if a.sourceMode == SourceStdin {
		a.results = append(a.results, CloneLogLine(line))
	}
	a.mu.Unlock()

	a.hub.BroadcastLine(line)
}

func (a *App) ApplyQuery(query string) {
	trimmed := strings.TrimSpace(query)
	var ast QueryNode
	if trimmed != "" {
		parsed, err := ParseQuery(trimmed)
		if err != nil {
			a.hub.BroadcastQueryError(*err)
			return
		}
		ast = parsed
	}

	a.mu.Lock()
	a.queryText = query
	a.queryAST = ast
	source := a.sourceMode
	a.mu.Unlock()

	a.hub.BroadcastSource(a.SourceState())
	if source == SourceAppInsights {
		a.QueryAzure()
		return
	}
	a.RefreshStdinResults()
}

func (a *App) SetTimeRange(timeRange TimeRange) {
	if _, _, err := timeRange.Resolve(time.Now()); err != nil {
		a.hub.BroadcastQueryError(QueryError{Message: err.Error(), Position: 0})
		return
	}

	a.mu.Lock()
	a.timeRange = timeRange
	source := a.sourceMode
	a.mu.Unlock()

	a.hub.BroadcastTimeRange(timeRange)
	if source == SourceAppInsights {
		a.QueryAzure()
		return
	}
	a.RefreshStdinResults()
}

func (a *App) SelectSource(source SourceMode) {
	if source == "" {
		source = SourceStdin
	}
	if source == SourceAppInsights && !a.appInsightsEnabled {
		a.hub.BroadcastQueryError(QueryError{Message: "Application Insights mode was not enabled; start sealog with --app-insights", Position: 0})
		return
	}
	if source != SourceStdin && source != SourceAppInsights {
		a.hub.BroadcastQueryError(QueryError{Message: "Unknown source", Position: 0})
		return
	}

	a.mu.Lock()
	a.sourceMode = source
	a.mu.Unlock()

	a.hub.BroadcastSource(a.SourceState())
	if source == SourceAppInsights {
		a.QueryAzure()
		return
	}
	a.RefreshStdinResults()
}

func (a *App) ListAzureResources() {
	a.setLoading(true, "Loading Application Insights resources")
	defer a.setLoading(false, "")

	resources, err := a.azure.ListAppInsightsResources(context.Background())
	errorMessage := ""
	if err != nil {
		errorMessage = azureErrorMessage(err)
		log.Printf("list Azure resources failed: %v", err)
	}

	a.mu.Lock()
	a.azureResources = resources
	a.azureError = errorMessage
	if a.selectedResource == nil && len(resources) == 1 {
		resource := resources[0]
		a.selectedResource = &resource
	}
	source := a.sourceStateLocked()
	a.mu.Unlock()

	a.hub.BroadcastAzureResources(resources, errorMessage)
	a.hub.BroadcastSource(source)
	if errorMessage == "" && len(resources) == 1 {
		a.QueryAzure()
	}
}

func (a *App) SelectAzureResource(resourceID string) {
	a.mu.Lock()
	var selected *AzureResource
	for _, resource := range a.azureResources {
		if resource.ID == resourceID {
			copied := resource
			selected = &copied
			break
		}
	}
	if selected == nil && resourceID != "" {
		selected = &AzureResource{ID: resourceID, Name: resourceID}
	}
	a.selectedResource = selected
	source := a.sourceStateLocked()
	a.mu.Unlock()

	a.hub.BroadcastSource(source)
	a.QueryAzure()
}

func (a *App) QueryAzure() {
	a.mu.RLock()
	if a.sourceMode != SourceAppInsights {
		a.mu.RUnlock()
		return
	}
	resource := a.selectedResource
	query := a.queryAST
	timeRange := a.timeRange
	a.mu.RUnlock()

	if resource == nil {
		a.mu.Lock()
		a.results = nil
		a.fields = nil
		a.total = 0
		a.mu.Unlock()
		a.hub.BroadcastFields(nil)
		a.hub.BroadcastResultsReset(0)
		return
	}

	a.setLoading(true, "Querying Application Insights")
	defer a.setLoading(false, "")

	lines, fields, err := a.azure.QueryTraces(context.Background(), *resource, query, timeRange)
	if err != nil {
		message := azureErrorMessage(err)
		log.Printf("query Azure traces failed: %v", err)
		a.hub.BroadcastQueryError(QueryError{Message: message, Position: 0})
		return
	}

	a.mu.Lock()
	a.results = cloneLines(lines)
	a.fields = fields
	a.total = len(lines)
	source := a.sourceStateLocked()
	a.mu.Unlock()

	a.hub.BroadcastSource(source)
	a.hub.BroadcastFields(fields)
	a.hub.BroadcastResultsReset(len(lines))
	for _, line := range lines {
		a.hub.BroadcastLine(line)
	}
}

func (a *App) RefreshStdinResults() {
	lines := a.buffer.Lines()
	fields := a.buffer.Fields()
	now := time.Now()

	a.mu.RLock()
	query := a.queryAST
	timeRange := a.timeRange
	a.mu.RUnlock()

	matches := make([]LogLine, 0, len(lines))
	for _, line := range lines {
		ok, err := lineMatches(line, query, timeRange, now)
		if err != nil {
			a.hub.BroadcastQueryError(QueryError{Message: err.Error(), Position: 0})
			return
		}
		if ok {
			matches = append(matches, line)
		}
	}

	a.mu.Lock()
	a.fields = fields
	a.results = cloneLines(matches)
	a.total = len(lines)
	total := a.total
	source := a.sourceStateLocked()
	a.mu.Unlock()

	a.hub.BroadcastSource(source)
	a.hub.BroadcastFields(fields)
	a.hub.BroadcastResultsReset(total)
	for _, line := range matches {
		a.hub.BroadcastLine(line)
	}
}

func (a *App) SourceState() SourceState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.sourceStateLocked()
}

func (a *App) sourceStateLocked() SourceState {
	var selected *AzureResource
	if a.selectedResource != nil {
		copied := *a.selectedResource
		selected = &copied
	}
	return SourceState{
		Mode:               a.sourceMode,
		AppInsightsEnabled: a.appInsightsEnabled,
		SelectedResource:   selected,
		Query:              a.queryText,
		Total:              a.total,
	}
}

func (a *App) setLoading(active bool, message string) {
	state := LoadingState{Active: active, Message: message}
	a.mu.Lock()
	a.loading = state
	a.mu.Unlock()
	a.hub.BroadcastLoading(state)
}

func lineMatches(line LogLine, query QueryNode, timeRange TimeRange, now time.Time) (bool, error) {
	inRange, err := timeRange.Contains(line.EventTime, now)
	if err != nil {
		return false, err
	}
	if !inRange {
		return false, nil
	}
	return EvaluateQuery(query, line.FlatFields), nil
}

func cloneLines(lines []LogLine) []LogLine {
	out := make([]LogLine, len(lines))
	for i, line := range lines {
		out[i] = CloneLogLine(line)
	}
	return out
}

func azureErrorMessage(err error) string {
	if authErr, ok := err.(*AzureAuthError); ok {
		return authErr.Message
	}
	return err.Error()
}
