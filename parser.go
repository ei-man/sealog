package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type LogLine struct {
	ID         int64          `json:"id"`
	Fields     map[string]any `json:"fields"`
	Raw        bool           `json:"raw"`
	FlatFields map[string]any `json:"-"`
	EventTime  time.Time      `json:"-"`
	IngestedAt time.Time      `json:"-"`
}

func ParseLine(line string) (map[string]any, bool) {
	data := []byte(line)
	if !json.Valid(data) {
		return map[string]any{"raw": line}, true
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return map[string]any{"raw": line}, true
	}

	if fields, ok := value.(map[string]any); ok {
		return fields, false
	}

	return map[string]any{"value": value}, false
}

func NewLogLine(id int64, fields map[string]any, raw bool, ingestedAt time.Time) LogLine {
	nested := cloneJSONMap(fields)
	nested["ingested_at"] = ingestedAt.UTC().Format(time.RFC3339Nano)
	flat := FlattenFields(nested)
	eventTime := DetectEventTime(flat, ingestedAt)

	return LogLine{
		ID:         id,
		Fields:     nested,
		Raw:        raw,
		FlatFields: flat,
		EventTime:  eventTime,
		IngestedAt: ingestedAt,
	}
}

func CloneLogLine(line LogLine) LogLine {
	line.Fields = cloneJSONMap(line.Fields)
	line.FlatFields = cloneFields(line.FlatFields)
	return line
}

func cloneJSONMap(fields map[string]any) map[string]any {
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = cloneJSONValue(value)
	}
	return out
}

func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneJSONMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneJSONValue(item)
		}
		return out
	default:
		return typed
	}
}

func FlattenFields(fields map[string]any) map[string]any {
	flat := make(map[string]any)
	flattenInto(flat, "", fields)
	return flat
}

func flattenInto(flat map[string]any, prefix string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 && prefix != "" {
			flat[prefix] = typed
			return
		}
		for key, nestedValue := range typed {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			flattenInto(flat, path, nestedValue)
		}
	default:
		if prefix != "" {
			flat[prefix] = typed
		}
	}
}

var timestampFieldOrder = []string{
	"timestamp",
	"@t",
	"Timestamp",
	"time",
	"Time",
	"datetime",
	"DateTime",
	"date",
	"Date",
	"eventTime",
	"event.time",
}

func DetectEventTime(flat map[string]any, fallback time.Time) time.Time {
	for _, field := range timestampFieldOrder {
		if value, ok := flat[field]; ok {
			if parsed, ok := parseLogTime(value); ok {
				return parsed
			}
		}
	}
	if value, ok := flat["ingested_at"]; ok {
		if parsed, ok := parseLogTime(value); ok {
			return parsed
		}
	}
	return fallback
}

func parseLogTime(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed, true
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return time.Time{}, false
		}
		if numeric, err := strconv.ParseFloat(text, 64); err == nil {
			return parseUnixNumber(numeric)
		}
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05.999999999",
			"2006-01-02T15:04:05.999",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02 15:04:05.999",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, layout := range layouts {
			if parsed, err := time.Parse(layout, text); err == nil {
				return parsed, true
			}
			if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
				return parsed, true
			}
		}
	case json.Number:
		if numeric, err := typed.Float64(); err == nil {
			return parseUnixNumber(numeric)
		}
	case float64:
		return parseUnixNumber(typed)
	case float32:
		return parseUnixNumber(float64(typed))
	case int:
		return parseUnixNumber(float64(typed))
	case int64:
		return parseUnixNumber(float64(typed))
	case int32:
		return parseUnixNumber(float64(typed))
	case uint:
		return parseUnixNumber(float64(typed))
	case uint64:
		return parseUnixNumber(float64(typed))
	case uint32:
		return parseUnixNumber(float64(typed))
	}
	return time.Time{}, false
}

func parseUnixNumber(value float64) (time.Time, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return time.Time{}, false
	}
	if math.Abs(value) < 10_000_000_000 {
		seconds := int64(value)
		nanos := int64((value - float64(seconds)) * 1_000_000_000)
		return time.Unix(seconds, nanos).UTC(), true
	}
	millis := int64(value)
	return time.UnixMilli(millis).UTC(), true
}

func valueToText(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case fmt.Stringer:
		return typed.String()
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		if bytes, err := json.Marshal(typed); err == nil {
			return string(bytes)
		}
		return fmt.Sprint(typed)
	}
}
