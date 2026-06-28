package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

type AzureResource struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ResourceGroup  string `json:"resourceGroup"`
	SubscriptionID string `json:"subscriptionId"`
	Location       string `json:"location"`
	AppID          string `json:"appId,omitempty"`
}

type AzureAuthError struct {
	Message string
}

func (e *AzureAuthError) Error() string {
	return e.Message
}

type AzureClient struct {
	httpClient *http.Client
}

func NewAzureClient() *AzureClient {
	return &AzureClient{
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *AzureClient) ListAppInsightsResources(ctx context.Context) ([]AzureResource, error) {
	output, err := runAzureCLI(ctx,
		"resource",
		"list",
		"--resource-type", "Microsoft.Insights/components",
		"-o", "json",
	)
	if err != nil {
		return nil, classifyAzureCLIError(err, output)
	}

	var rawResources []struct {
		ID             string         `json:"id"`
		Name           string         `json:"name"`
		ResourceGroup  string         `json:"resourceGroup"`
		SubscriptionID string         `json:"subscriptionId"`
		Location       string         `json:"location"`
		Properties     map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(output, &rawResources); err != nil {
		return nil, fmt.Errorf("parse Azure resource list: %w", err)
	}

	resources := make([]AzureResource, 0, len(rawResources))
	for _, raw := range rawResources {
		resource := AzureResource{
			ID:             raw.ID,
			Name:           raw.Name,
			ResourceGroup:  raw.ResourceGroup,
			SubscriptionID: raw.SubscriptionID,
			Location:       raw.Location,
			AppID:          stringProperty(raw.Properties, "AppId", "appId"),
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func (c *AzureClient) QueryTraces(ctx context.Context, resource AzureResource, query QueryNode, timeRange TimeRange) ([]LogLine, []string, error) {
	if resource.AppID == "" {
		appID, err := c.lookupAppID(ctx, resource.ID)
		if err != nil {
			return nil, nil, err
		}
		resource.AppID = appID
	}
	if resource.AppID == "" {
		return nil, nil, fmt.Errorf("Application Insights resource %q does not expose an app id", resource.Name)
	}

	start, end, err := timeRange.Resolve(time.Now())
	if err != nil {
		return nil, nil, err
	}

	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, nil, err
	}

	kql := BuildTracesKQL(query, start, end)
	requestBody, err := json.Marshal(map[string]string{"query": kql})
	if err != nil {
		return nil, nil, err
	}

	endpoint := "https://api.applicationinsights.io/v1/apps/" + url.PathEscape(resource.AppID) + "/query"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("query Application Insights traces: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read Application Insights response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, nil, &AzureAuthError{Message: "Azure authentication failed; run `az login`"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("Application Insights query failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	lines, fields, err := parseAppInsightsResponse(body)
	if err != nil {
		return nil, nil, err
	}
	return lines, fields, nil
}

func (c *AzureClient) lookupAppID(ctx context.Context, resourceID string) (string, error) {
	if resourceID == "" {
		return "", fmt.Errorf("missing Application Insights resource id")
	}
	output, err := runAzureCLI(ctx,
		"resource",
		"show",
		"--ids", resourceID,
		"--query", "properties.AppId",
		"-o", "tsv",
	)
	if err != nil {
		return "", classifyAzureCLIError(err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

func (c *AzureClient) accessToken(ctx context.Context) (string, error) {
	output, err := runAzureCLI(ctx,
		"account",
		"get-access-token",
		"--resource", "https://api.applicationinsights.io",
		"--query", "accessToken",
		"-o", "tsv",
	)
	if err != nil {
		return "", classifyAzureCLIError(err, output)
	}
	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", &AzureAuthError{Message: "Azure authentication failed; run `az login`"}
	}
	return token, nil
}

func runAzureCLI(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "az", args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.Bytes(), err
}

func classifyAzureCLIError(err error, output []byte) error {
	text := strings.TrimSpace(string(output))
	lower := strings.ToLower(text)
	if errors.Is(err, exec.ErrNotFound) || strings.Contains(lower, "az: command not found") {
		return &AzureAuthError{Message: "Azure CLI is required; install `az` and run `az login`"}
	}
	if strings.Contains(lower, "az login") ||
		strings.Contains(lower, "please run") ||
		strings.Contains(lower, "login required") ||
		strings.Contains(lower, "not logged") ||
		strings.Contains(lower, "expired") {
		return &AzureAuthError{Message: "Azure authentication failed; run `az login`"}
	}
	if text != "" {
		return fmt.Errorf("%w: %s", err, text)
	}
	return err
}

func stringProperty(properties map[string]any, names ...string) string {
	for _, name := range names {
		if value, ok := properties[name]; ok {
			return valueToText(value)
		}
	}
	return ""
}

func BuildTracesKQL(query QueryNode, start time.Time, end time.Time) string {
	var builder strings.Builder
	builder.WriteString("traces\n")
	builder.WriteString("| where timestamp >= datetime(")
	builder.WriteString(start.UTC().Format(time.RFC3339Nano))
	builder.WriteString(") and timestamp <= datetime(")
	builder.WriteString(end.UTC().Format(time.RFC3339Nano))
	builder.WriteString(")\n")
	if queryKQL := QueryToKQL(query); queryKQL != "" {
		builder.WriteString("| where ")
		builder.WriteString(queryKQL)
		builder.WriteString("\n")
	}
	builder.WriteString("| order by timestamp asc\n")
	builder.WriteString("| take 5000\n")
	builder.WriteString("| project timestamp, message, severityLevel, operation_Id, operation_Name, operation_ParentId, cloud_RoleName, cloud_RoleInstance, client_Type, client_City, client_StateOrProvince, client_CountryOrRegion, customDimensions")
	return builder.String()
}

type appInsightsResponse struct {
	Tables []struct {
		Name    string `json:"name"`
		Columns []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"columns"`
		Rows [][]any `json:"rows"`
	} `json:"tables"`
}

func parseAppInsightsResponse(body []byte) ([]LogLine, []string, error) {
	var parsed appInsightsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, nil, fmt.Errorf("parse Application Insights response: %w", err)
	}
	if len(parsed.Tables) == 0 {
		return nil, nil, nil
	}

	table := parsed.Tables[0]
	fieldsSeen := make(map[string]struct{})
	lines := make([]LogLine, 0, len(table.Rows))
	for rowIndex, row := range table.Rows {
		fields := make(map[string]any)
		for columnIndex, column := range table.Columns {
			if columnIndex >= len(row) {
				continue
			}
			value := row[columnIndex]
			if value == nil {
				continue
			}
			if column.Name == "customDimensions" {
				mergeCustomDimensions(fields, value)
				continue
			}
			fields[column.Name] = value
		}
		line := NewLogLine(int64(rowIndex+1), fields, false, time.Now())
		for field := range line.FlatFields {
			fieldsSeen[field] = struct{}{}
		}
		lines = append(lines, line)
	}

	return lines, sortedFields(fieldsSeen), nil
}

func mergeCustomDimensions(fields map[string]any, value any) {
	dimensions := customDimensionsMap(value)
	for key, dimensionValue := range dimensions {
		fields[key] = parseJSONLookingValue(dimensionValue)
	}
}

func customDimensionsMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case string:
		var decoded map[string]any
		decoder := json.NewDecoder(strings.NewReader(typed))
		decoder.UseNumber()
		if err := decoder.Decode(&decoded); err == nil {
			return decoded
		}
	}
	return nil
}

func parseJSONLookingValue(value any) any {
	text, ok := value.(string)
	if !ok {
		return value
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return value
	}
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return value
	}

	var decoded any
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return value
	}
	return decoded
}
