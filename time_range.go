package main

import (
	"fmt"
	"time"
)

const defaultTimeRangePreset = "1h"

type TimeRange struct {
	Preset string     `json:"preset"`
	Start  *time.Time `json:"start,omitempty"`
	End    *time.Time `json:"end,omitempty"`
}

func DefaultTimeRange() TimeRange {
	return TimeRange{Preset: defaultTimeRangePreset}
}

func (r TimeRange) Resolve(now time.Time) (time.Time, time.Time, error) {
	preset := r.Preset
	if preset == "" {
		preset = defaultTimeRangePreset
	}

	if preset == "custom" {
		if r.Start == nil || r.End == nil {
			return time.Time{}, time.Time{}, fmt.Errorf("custom time range requires both start and end time")
		}
		start := r.Start.UTC()
		end := r.End.UTC()
		if !start.Before(end) {
			return time.Time{}, time.Time{}, fmt.Errorf("custom time range start must be before end")
		}
		return start, end, nil
	}

	duration, ok := presetDurations[preset]
	if !ok {
		return time.Time{}, time.Time{}, fmt.Errorf("unknown time range preset %q", preset)
	}

	end := now.UTC()
	start := end.Add(-duration)
	return start, end, nil
}

var presetDurations = map[string]time.Duration{
	"15m": 15 * time.Minute,
	"30m": 30 * time.Minute,
	"1h":  time.Hour,
	"3h":  3 * time.Hour,
	"6h":  6 * time.Hour,
	"12h": 12 * time.Hour,
	"24h": 24 * time.Hour,
	"3d":  72 * time.Hour,
}

func (r TimeRange) Contains(t time.Time, now time.Time) (bool, error) {
	start, end, err := r.Resolve(now)
	if err != nil {
		return false, err
	}
	target := t.UTC()
	return !target.Before(start) && !target.After(end), nil
}
