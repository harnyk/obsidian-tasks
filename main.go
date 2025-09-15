package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/teambition/rrule-go"
	"gopkg.in/yaml.v3"
)

type FrontMatter struct {
	RRule    string   `yaml:"rrule"`
	Duration string   `yaml:"duration"`
	DTStart  string   `yaml:"dtstart"`
	Tags     []string `yaml:"tags"`
}

type Task struct {
	Name     string
	RRule    string
	Duration string
}

type Config struct {
	NotesDir string `yaml:"notes_dir"`
}

func getNotesDir() string {
	// Try environment variable first
	if root := os.Getenv("OBSIDIAN_NOTES_DIR"); root != "" {
		return root
	}

	// Try config files in order of preference
	homeDir, _ := os.UserHomeDir()
	configPaths := []string{
		"config.yaml",
		"config.yml",
		filepath.Join(homeDir, ".config", "obsidian-tasks", "config.yaml"),
		filepath.Join(homeDir, ".config", "obsidian-tasks", "config.yml"),
	}

	for _, configPath := range configPaths {
		if data, err := os.ReadFile(configPath); err == nil {
			var config Config
			if err := yaml.Unmarshal(data, &config); err == nil && config.NotesDir != "" {
				return config.NotesDir
			}
		}
	}

	fmt.Println("Error: Notes directory not configured. Set OBSIDIAN_NOTES_DIR environment variable or create config.yaml with notes_dir field")
	os.Exit(1)
	return ""
}

func main() {
	root := getNotesDir()

	var activeTasks []Task
	var inactiveTasks []Task

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			if task := processFile(path); task.Name != "" {
				if isTaskActive(path) {
					activeTasks = append(activeTasks, task)
				} else {
					inactiveTasks = append(inactiveTasks, task)
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Walk error:", err)
		return
	}

	printTasks("Active tasks", activeTasks, color.FgGreen)
	printTasks("Inactive tasks", inactiveTasks, color.FgHiBlack)
}

func printTasks(title string, tasks []Task, nameColor color.Attribute) {
	if len(tasks) == 0 {
		return
	}
	color.New(color.FgYellow, color.Bold).Println("\n" + title + ":")
	for _, task := range tasks {
		fmt.Print("  - ")
		color.New(nameColor, color.Bold).Print(task.Name)
		color.New(color.Reset).Print(" (" + task.RRule)
		if task.Duration != "" {
			color.New(color.Reset).Print(", " + task.Duration)
		}
		color.New(color.Reset).Println(")")
	}
}

func parseFrontMatter(path string) (*FrontMatter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("no frontmatter")
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid frontmatter format")
	}

	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return nil, fmt.Errorf("YAML parsing error: %w", err)
	}

	return &fm, nil
}

func parseDuration(durationStr string) (time.Duration, error) {
	if durationStr == "" {
		return 24 * time.Hour, nil // Default to 1 day
	}

	// Parse ISO 8601 duration format (P1D, P1W, P1M, PT1H, etc.)
	if !strings.HasPrefix(durationStr, "P") {
		return 0, fmt.Errorf("duration must start with 'P'")
	}

	duration := time.Duration(0)
	remaining := durationStr[1:] // Remove 'P'

	// Check for time component (after 'T')
	timePart := ""
	if tIndex := strings.Index(remaining, "T"); tIndex >= 0 {
		timePart = remaining[tIndex+1:]
		remaining = remaining[:tIndex]
	}

	// Parse date components (before 'T')
	for remaining != "" {
		i := 0
		for i < len(remaining) && (remaining[i] >= '0' && remaining[i] <= '9') {
			i++
		}
		if i == 0 {
			break
		}

		value := remaining[:i]
		unit := remaining[i:i+1]
		remaining = remaining[i+1:]

		num, err := time.ParseDuration(value + "h")
		if err != nil {
			return 0, err
		}
		hours := int(num.Hours())

		switch unit {
		case "D":
			duration += time.Duration(hours) * 24 * time.Hour
		case "W":
			duration += time.Duration(hours) * 7 * 24 * time.Hour
		case "M":
			duration += time.Duration(hours) * 30 * 24 * time.Hour // Approximate
		case "Y":
			duration += time.Duration(hours) * 365 * 24 * time.Hour // Approximate
		default:
			return 0, fmt.Errorf("unknown date unit: %s", unit)
		}
	}

	// Parse time components (after 'T')
	for timePart != "" {
		i := 0
		for i < len(timePart) && (timePart[i] >= '0' && timePart[i] <= '9') {
			i++
		}
		if i == 0 {
			break
		}

		value := timePart[:i]
		unit := timePart[i:i+1]
		timePart = timePart[i+1:]

		switch unit {
		case "H":
			if hours, err := time.ParseDuration(value + "h"); err == nil {
				duration += hours
			}
		case "M":
			if minutes, err := time.ParseDuration(value + "m"); err == nil {
				duration += minutes
			}
		case "S":
			if seconds, err := time.ParseDuration(value + "s"); err == nil {
				duration += seconds
			}
		default:
			return 0, fmt.Errorf("unknown time unit: %s", unit)
		}
	}

	return duration, nil
}

func parseStartDate(dtStartStr string) time.Time {
	if dtStartStr == "" {
		// Default to 1 year ago
		return time.Now().AddDate(-1, 0, 0).Truncate(24 * time.Hour)
	}

	// Try parsing common date formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"20060102T000000Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dtStartStr); err == nil {
			return t.Truncate(24 * time.Hour)
		}
	}

	// If parsing fails, default to 1 year ago
	return time.Now().AddDate(-1, 0, 0).Truncate(24 * time.Hour)
}

func processFile(path string) Task {
	fm, err := parseFrontMatter(path)
	if err != nil {
		if !strings.Contains(err.Error(), "no frontmatter") {
			fmt.Println("Error processing", path+":", err)
		}
		return Task{}
	}

	if fm.RRule != "" {
		filename := cleanFilename(filepath.Base(path))
		return Task{Name: filename, RRule: fm.RRule, Duration: fm.Duration}
	}
	return Task{}
}

func isTaskActive(path string) bool {
	fm, err := parseFrontMatter(path)
	if err != nil {
		return false
	}

	if fm.RRule == "" {
		return false
	}

	today := time.Now().Truncate(24 * time.Hour)
	startDate := parseStartDate(fm.DTStart)
	duration, err := parseDuration(fm.Duration)
	if err != nil {
		return false
	}

	// Create RRULE with proper DTSTART
	r, err := rrule.StrToRRule("DTSTART:" + startDate.Format("20060102T000000Z") + "\nRRULE:" + fm.RRule)
	if err != nil {
		return false
	}

	// Get all occurrences from start date to today + duration
	// (we need to check a bit into the future in case an occurrence + duration overlaps with today)
	endDate := today.Add(duration)
	occurrences := r.Between(startDate, endDate, true)

	// Check if today falls within any occurrence's active window
	for _, occurrence := range occurrences {
		occurrenceStart := occurrence.Truncate(24 * time.Hour)
		occurrenceEnd := occurrenceStart.Add(duration)

		if (today.Equal(occurrenceStart) || today.After(occurrenceStart)) && today.Before(occurrenceEnd) {
			return true
		}
	}

	return false
}

func cleanFilename(filename string) string {
	// Remove date prefixes like "2025-05-22 ", "2025-05-22_", "2025.05.22 ", etc.
	datePattern := regexp.MustCompile(`^(\d{4}[-_.]\d{1,2}[-_.]\d{1,2}[\s_-]*)+`)
	cleaned := datePattern.ReplaceAllString(filename, "")
	cleaned = strings.TrimSuffix(cleaned, ".md")

	return cleaned
}
