package main

import (
	"strings"
	"testing"
	"time"
)

func TestQueryEvaluateExactRegexBooleanAndDotPath(t *testing.T) {
	fields := FlattenFields(map[string]any{
		"level":   "error",
		"message": "request timeout",
		"person": map[string]any{
			"name": "Ada",
		},
	})

	query, err := ParseQuery(`level=error AND (message~="timeout" OR person.name="Grace")`)
	if err != nil {
		t.Fatalf("ParseQuery returned error: %v", err)
	}
	if !EvaluateQuery(query, fields) {
		t.Fatal("expected query to match nested fields")
	}

	query, err = ParseQuery(`NOT person.name="Grace" AND level="ERROR"`)
	if err != nil {
		t.Fatalf("ParseQuery returned error: %v", err)
	}
	if !EvaluateQuery(query, fields) {
		t.Fatal("expected NOT and case-insensitive exact match to pass")
	}
}

func TestQueryRequiresExplicitBooleanOperators(t *testing.T) {
	_, err := ParseQuery(`level=error message~="timeout"`)
	if err == nil {
		t.Fatal("expected adjacent conditions to fail")
	}
	if err.Message != "Unexpected token after query" {
		t.Fatalf("unexpected error message: %q", err.Message)
	}
}

func TestQueryMissingFieldAndInvalidRegex(t *testing.T) {
	query, err := ParseQuery(`missing=value`)
	if err != nil {
		t.Fatalf("ParseQuery returned error: %v", err)
	}
	if EvaluateQuery(query, map[string]any{"level": "error"}) {
		t.Fatal("missing field should evaluate false")
	}

	_, err = ParseQuery(`message~="["`)
	if err == nil {
		t.Fatal("expected invalid regex to fail")
	}
}

func TestLineMatchesUsesEventTimestampBeforeIngestedAt(t *testing.T) {
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	eventTime := now.Add(-30 * time.Minute)
	ingestedAt := now.Add(-6 * time.Hour)
	line := NewLogLine(1, map[string]any{
		"timestamp": eventTime.Format(time.RFC3339Nano),
		"level":     "info",
	}, false, ingestedAt)

	ok, err := lineMatches(line, nil, TimeRange{Preset: "1h"}, now)
	if err != nil {
		t.Fatalf("lineMatches returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected event timestamp to keep line in range")
	}
}

func TestLineMatchesFallsBackToIngestedAt(t *testing.T) {
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	line := NewLogLine(1, map[string]any{"message": "no event timestamp"}, false, now.Add(-20*time.Minute))

	ok, err := lineMatches(line, nil, TimeRange{Preset: "30m"}, now)
	if err != nil {
		t.Fatalf("lineMatches returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected ingested_at fallback to keep line in range")
	}
}

func TestCustomTimeRangeValidation(t *testing.T) {
	_, _, err := (TimeRange{Preset: "custom"}).Resolve(time.Now())
	if err == nil {
		t.Fatal("expected missing custom start/end to fail")
	}
}

func TestBuildTracesKQLUsesTimeBoundsAndCustomDimensions(t *testing.T) {
	query, err := ParseQuery(`message~="timeout" AND person.name="Ada"`)
	if err != nil {
		t.Fatalf("ParseQuery returned error: %v", err)
	}
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)
	kql := BuildTracesKQL(query, start, end)

	expected := []string{
		"timestamp >= datetime(2026-06-28T10:00:00Z)",
		"timestamp <= datetime(2026-06-28T11:00:00Z)",
		"message matches regex '(?i)timeout'",
		`parse_json(tostring(customDimensions['person']))['name']`,
		"=~ 'Ada'",
	}
	for _, needle := range expected {
		if !strings.Contains(kql, needle) {
			t.Fatalf("KQL missing %q:\n%s", needle, kql)
		}
	}
}

func TestParseAppInsightsResponseExpandsCustomDimensionsJSON(t *testing.T) {
	body := []byte(`{
		"tables": [{
			"name": "PrimaryResult",
			"columns": [
				{"name": "timestamp", "type": "datetime"},
				{"name": "message", "type": "string"},
				{"name": "customDimensions", "type": "dynamic"}
			],
			"rows": [[
				"2026-06-28T10:15:00Z",
				"hello",
				{"person": "{\"name\":\"Ada\"}", "tenant": "alpha"}
			]]
		}]
	}`)

	lines, fields, err := parseAppInsightsResponse(body)
	if err != nil {
		t.Fatalf("parseAppInsightsResponse returned error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if got := lines[0].FlatFields["person.name"]; got != "Ada" {
		t.Fatalf("expected nested custom dimension, got %#v", got)
	}
	if !contains(fields, "person.name") || !contains(fields, "tenant") {
		t.Fatalf("expected flattened fields, got %#v", fields)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
