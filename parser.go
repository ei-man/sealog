package main

import (
	"bytes"
	"encoding/json"
)

type LogLine struct {
	ID     int64          `json:"id"`
	Fields map[string]any `json:"fields"`
	Raw    bool           `json:"raw"`
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
