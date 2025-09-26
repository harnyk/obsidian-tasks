package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsTaskActive(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Test assumes current date is Friday, September 26, 2025 (as in current output)

	tests := []struct {
		name        string
		frontMatter string
		expected    bool
		description string
	}{
		{
			name: "monthly_task_active_1",
			frontMatter: `---
rrule: FREQ=MONTHLY;BYMONTHDAY=20
duration: P10D
dtstart: 2024-01-20
---`,
			expected:    true,
			description: "Monthly task on 20th with 10-day duration should be active on Sep 26",
		},
		{
			name: "monthly_task_active_2",
			frontMatter: `---
rrule: FREQ=MONTHLY;BYMONTHDAY=-5
duration: P5D
dtstart: 2024-01-26
---`,
			expected:    true,
			description: "Monthly task on last 5th day with 5-day duration should be active on Sep 26",
		},
		{
			name: "monthly_task_inactive_1",
			frontMatter: `---
rrule: FREQ=MONTHLY;BYMONTHDAY=12
duration: P6D
dtstart: 2024-01-12
---`,
			expected:    false,
			description: "Monthly task on 12th with 6-day duration should be inactive on Sep 26",
		},
		{
			name: "monthly_task_inactive_2",
			frontMatter: `---
rrule: FREQ=MONTHLY;BYMONTHDAY=1
dtstart: 2024-01-01
---`,
			expected:    false,
			description: "Monthly task on 1st with default duration should be inactive on Sep 26",
		},
		{
			name: "monthly_task_inactive_3",
			frontMatter: `---
rrule: FREQ=MONTHLY;BYMONTHDAY=1
duration: P3D
dtstart: 2024-01-01
---`,
			expected:    false,
			description: "Monthly task on 1st with 3-day duration should be inactive on Sep 26",
		},
		{
			name: "weekly_task_should_be_active",
			frontMatter: `---
rrule: FREQ=WEEKLY;BYDAY=FR
dtstart: 2024-01-05
---`,
			expected:    true,
			description: "Weekly Friday task should be active on Friday Sep 26",
		},
		{
			name: "one_time_task_inactive",
			frontMatter: `---
dtstart: 2025-10-18
duration: P6D
---`,
			expected:    false,
			description: "One-time task starting Oct 18 should be inactive on Sep 26",
		},
	}

	// Note: These tests assume the current date is Friday, September 26, 2025
	// as shown in the actual program output

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test file
			testFile := filepath.Join(tempDir, tt.name+".md")
			err := os.WriteFile(testFile, []byte(tt.frontMatter), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test the function
			result, err := isTaskActive(testFile)
			if err != nil && tt.expected {
				t.Errorf("%s: unexpected error: %v - %s", tt.name, err, tt.description)
			}
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v - %s", tt.name, tt.expected, result, tt.description)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"", 24 * time.Hour, false},          // Default 1 day
		{"P1D", 24 * time.Hour, false},       // 1 day
		{"P10D", 10 * 24 * time.Hour, false}, // 10 days
		{"P5D", 5 * 24 * time.Hour, false},   // 5 days
		{"P6D", 6 * 24 * time.Hour, false},   // 6 days
		{"P3D", 3 * 24 * time.Hour, false},   // 3 days
		{"P1W", 7 * 24 * time.Hour, false},   // 1 week
		{"PT2H", 2 * time.Hour, false},       // 2 hours
		{"PT30M", 30 * time.Minute, false},   // 30 minutes
		{"P1DT2H", 26 * time.Hour, false},    // 1 day + 2 hours
		{"invalid", 0, true},                 // Invalid format
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseDuration(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %q, got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("For input %q: expected %v, got %v", tt.input, tt.expected, result)
				}
			}
		})
	}
}

func TestParseStartDate(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"2024-01-20", time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)},
		{"2024-01-26", time.Date(2024, 1, 26, 0, 0, 0, 0, time.UTC)},
		{"2024-01-12", time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC)},
		{"2024-01-01", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"2024-01-05", time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)},
		{"2025-10-18", time.Date(2025, 10, 18, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseStartDate(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("For input %q: expected %v, got %v", tt.input, tt.expected, result)
			}
		})
	}
}

func TestParseFrontMatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    *FrontMatter
		expectError bool
	}{
		{
			name: "valid_frontmatter",
			content: `---
rrule: FREQ=WEEKLY;BYDAY=FR
duration: P1D
dtstart: 2024-01-05
---

# Task content`,
			expected: &FrontMatter{
				RRule:    "FREQ=WEEKLY;BYDAY=FR",
				Duration: "P1D",
				DTStart:  "2024-01-05",
				Tags:     nil,
			},
			expectError: false,
		},
		{
			name:        "no_frontmatter",
			content:     "# Regular markdown file",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFrontMatter(tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.RRule != tt.expected.RRule {
				t.Errorf("RRule: expected %q, got %q", tt.expected.RRule, result.RRule)
			}
		})
	}
}

func TestPipeline_Integration(t *testing.T) {
	// Test the full pipeline: ParseFrontMatter -> ApplyDefaults -> IsTaskActive
	currentTime := time.Date(2025, 9, 26, 12, 0, 0, 0, time.UTC) // Friday, Sep 26, 2025

	content := `---
rrule: FREQ=WEEKLY;BYDAY=FR
duration: P1D
dtstart: 2024-01-05
---

# Weekly Friday Task`

	// Step 1: Parse
	fm, err := ParseFrontMatter(content)
	if err != nil {
		t.Fatalf("ParseFrontMatter failed: %v", err)
	}

	// Step 2: Apply defaults
	fmWithDefaults, err := ApplyDefaults(fm, currentTime)
	if err != nil {
		t.Fatalf("ApplyDefaults failed: %v", err)
	}

	// Step 3: Check if active
	isActive, err := IsTaskActive(fmWithDefaults, currentTime)
	if err != nil {
		t.Fatalf("IsTaskActive failed: %v", err)
	}

	// Should be active on Friday
	if !isActive {
		t.Errorf("Expected Friday task to be active on Friday, but got false")
	}
}
