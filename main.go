package main

import (
	"fmt"
	"io/fs"
	"net/url"
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

type FrontMatterWithDefaults struct {
	RRule    string
	Duration time.Duration
	DTStart  time.Time
	Tags     []string
}

type Task struct {
	Name      string
	RRule     string
	Duration  string
	NextStart *time.Time
	DueDate   *time.Time
	Error     error
	FilePath  string
}

type Config struct {
	NotesDir string `yaml:"notes_dir"`
}

type VaultInfo struct {
	Name string
	Path string
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

func detectVault(notesDir string) *VaultInfo {
	currentPath := notesDir

	for {
		// Check if .obsidian folder exists in current directory
		obsidianPath := filepath.Join(currentPath, ".obsidian")
		if _, err := os.Stat(obsidianPath); err == nil {
			// Found .obsidian folder, extract vault name from directory name
			vaultName := filepath.Base(currentPath)
			return &VaultInfo{
				Name: vaultName,
				Path: currentPath,
			}
		}

		// Move up one directory
		parentPath := filepath.Dir(currentPath)

		// If we've reached the root or can't go further up, stop
		if parentPath == currentPath || parentPath == "/" || parentPath == "." {
			break
		}

		currentPath = parentPath
	}

	return nil
}

func createObsidianURI(vaultName, filePath, vaultPath, notesDir string) string {
	// Calculate relative path from vault root to the file
	relativeFilePath, _ := filepath.Rel(vaultPath, filePath)

	// Remove .md extension and convert to forward slashes
	relativeFilePath = strings.TrimSuffix(relativeFilePath, ".md")
	relativeFilePath = strings.ReplaceAll(relativeFilePath, "\\", "/")

	// URL encode the components (using %20 for spaces, not +)
	encodedVault := url.PathEscape(vaultName)
	encodedFile := url.PathEscape(relativeFilePath)

	return fmt.Sprintf("obsidian://open?vault=%s&file=%s", encodedVault, encodedFile)
}

func createTerminalHyperlink(uri, text string) string {
	// OSC 8 escape sequence format: \x1b]8;;URI\x1b\\TEXT\x1b]8;;\x1b\\
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", uri, text)
}

func main() {
	// Check for help flag
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printHelp()
		return
	}

	root := getNotesDir()

	// Detect Obsidian vault
	vault := detectVault(root)
	if vault != nil {
		color.New(color.FgCyan, color.Bold).Printf("📓 Vault: %s\n", vault.Name)
	}

	var activeTasks []Task
	var inactiveTasks []Task
	var errorTasks []Task

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			if task := processFile(path); task.Name != "" {
				active, taskErr := isTaskActive(path)
				if taskErr != nil {
					task.Error = taskErr
					errorTasks = append(errorTasks, task)
				} else if active {
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

	printTasks("Active tasks", activeTasks, color.FgGreen, vault, root)
	printTasks("Inactive tasks", inactiveTasks, color.FgHiBlack, vault, root)
	printTasksWithErrors("Tasks with syntax errors", errorTasks, color.FgRed, vault, root)
}

func printHelp() {
	fmt.Println("obsidian-tasks - CLI tool for managing recurring tasks in Obsidian notes")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  obsidian-tasks [--help]")
	fmt.Println()
	fmt.Println("DESCRIPTION:")
	fmt.Println("  Scans Obsidian markdown files for recurring tasks defined with iCal RRULE + DURATION")
	fmt.Println("  semantics in YAML front matter. Displays active and inactive tasks with smart")
	fmt.Println("  date indicators including due dates and next start dates.")
	fmt.Println()
	fmt.Println("CONFIGURATION:")
	fmt.Println("  Set notes directory via:")
	fmt.Println("  - OBSIDIAN_NOTES_DIR environment variable, or")
	fmt.Println("  - Config file (config.yaml/config.yml) with 'notes_dir' field in:")
	fmt.Println("    - Current directory")
	fmt.Println("    - ~/.config/obsidian-tasks/")
	fmt.Println()
	fmt.Println("FRONT MATTER FORMAT:")
	fmt.Println("  Recurring tasks:")
	fmt.Println("    ---")
	fmt.Println("    rrule: FREQ=DAILY;COUNT=5")
	fmt.Println("    duration: P1D")
	fmt.Println("    dtstart: 2025-01-01")
	fmt.Println("    ---")
	fmt.Println()
	fmt.Println("  One-time events:")
	fmt.Println("    ---")
	fmt.Println("    dtstart: 2025-10-18")
	fmt.Println("    duration: P6D")
	fmt.Println("    ---")
	fmt.Println()
	fmt.Println("DURATION FORMAT:")
	fmt.Println("  ISO 8601 duration: P1D (1 day), P1W (1 week), PT2H (2 hours), etc.")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -h, --help    Show this help message")
}

func printTasks(title string, tasks []Task, nameColor color.Attribute, vault *VaultInfo, notesDir string) {
	if len(tasks) == 0 {
		return
	}
	color.New(color.FgYellow, color.Bold).Println("\n" + title + ":")
	for _, task := range tasks {
		fmt.Print("  - ")

		// Create hyperlink if vault is available
		if vault != nil && task.FilePath != "" {
			uri := createObsidianURI(vault.Name, task.FilePath, vault.Path, notesDir)
			hyperlinkText := createTerminalHyperlink(uri, task.Name)
			color.New(nameColor, color.Bold).Print(hyperlinkText)
		} else {
			color.New(nameColor, color.Bold).Print(task.Name)
		}
		color.New(color.Reset).Print(" (" + task.RRule)
		if task.Duration != "" {
			color.New(color.Reset).Print(", " + task.Duration)
		}

		// Show due date for active tasks
		if nameColor == color.FgGreen && task.DueDate != nil {
			today := time.Now().Truncate(24 * time.Hour)
			dateStr := task.DueDate.Format("2006-01-02")

			if task.DueDate.Equal(today) {
				// Red highlight if due today
				color.New(color.FgRed, color.Bold).Print(" ⚠️ " + dateStr)
			} else {
				// Normal color for future due dates
				color.New(color.FgYellow).Print(" → " + dateStr)
			}
		}

		// Show next start date for inactive tasks
		if nameColor == color.FgHiBlack && task.NextStart != nil {
			color.New(color.FgCyan).Print(" → " + task.NextStart.Format("2006-01-02"))
		}

		color.New(color.Reset).Println(")")
	}
}

func printTasksWithErrors(title string, tasks []Task, nameColor color.Attribute, vault *VaultInfo, notesDir string) {
	if len(tasks) == 0 {
		return
	}
	color.New(color.FgYellow, color.Bold).Println("\n" + title + ":")
	for _, task := range tasks {
		fmt.Print("  - ")

		// Create hyperlink if vault is available
		if vault != nil && task.FilePath != "" {
			uri := createObsidianURI(vault.Name, task.FilePath, vault.Path, notesDir)
			hyperlinkText := createTerminalHyperlink(uri, task.Name)
			color.New(nameColor, color.Bold).Print(hyperlinkText)
		} else {
			color.New(nameColor, color.Bold).Print(task.Name)
		}
		color.New(color.Reset).Print(" (" + task.RRule)
		if task.Duration != "" {
			color.New(color.Reset).Print(", " + task.Duration)
		}
		color.New(color.Reset).Print(")")

		// Show error message
		if task.Error != nil {
			color.New(color.FgRed).Print(" ❌ " + task.Error.Error())
		}

		fmt.Println()
	}
}

// ParseFrontMatter parses YAML frontmatter from content string
func ParseFrontMatter(content string) (*FrontMatter, error) {
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

// parseFrontMatter reads file and parses frontmatter (wrapper for file I/O)
func parseFrontMatter(path string) (*FrontMatter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	return ParseFrontMatter(string(data))
}

// ParseDuration parses ISO 8601 duration string
func ParseDuration(durationStr string) (time.Duration, error) {
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
		unit := remaining[i : i+1]
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
		unit := timePart[i : i+1]
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

func getNextOccurrence(fm *FrontMatter) *time.Time {
	if fm.RRule == "" {
		return nil
	}

	today := time.Now().Truncate(24 * time.Hour)
	startDate := parseStartDate(fm.DTStart)

	r, err := rrule.StrToRRule("DTSTART:" + startDate.Format("20060102T000000Z") + "\nRRULE:" + fm.RRule)
	if err != nil {
		return nil
	}

	// Get next occurrence after today
	nextOccurrences := r.Between(today.Add(24*time.Hour), today.AddDate(1, 0, 0), true)
	if len(nextOccurrences) > 0 {
		next := nextOccurrences[0].Truncate(24 * time.Hour)
		return &next
	}

	return nil
}

func getCurrentDueDate(fm *FrontMatter) *time.Time {
	if fm.RRule == "" {
		return nil
	}

	today := time.Now().Truncate(24 * time.Hour)
	startDate := parseStartDate(fm.DTStart)
	duration, err := ParseDuration(fm.Duration)
	if err != nil {
		return nil
	}

	r, err := rrule.StrToRRule("DTSTART:" + startDate.Format("20060102T000000Z") + "\nRRULE:" + fm.RRule)
	if err != nil {
		return nil
	}

	// Find current active occurrence and its due date
	endDate := today.Add(duration)
	occurrences := r.Between(startDate, endDate, true)

	for _, occurrence := range occurrences {
		occurrenceStart := occurrence.Truncate(24 * time.Hour)
		occurrenceEnd := occurrenceStart.Add(duration)

		// If today falls within this occurrence's window, return its due date
		if (today.Equal(occurrenceStart) || today.After(occurrenceStart)) && today.Before(occurrenceEnd) {
			dueDate := occurrenceEnd.Add(-24 * time.Hour) // Last day of active period
			return &dueDate
		}
	}

	return nil
}

func getOneTimeDueDate(fm *FrontMatter) *time.Time {
	if fm.DTStart == "" {
		return nil
	}

	startDate := parseStartDate(fm.DTStart)
	duration, err := ParseDuration(fm.Duration)
	if err != nil {
		return nil
	}

	dueDate := startDate.Add(duration).Add(-24 * time.Hour) // Last day of active period
	return &dueDate
}

// IsOneTimeTaskActive checks if one-time task is active at given time
func IsOneTimeTaskActive(fm *FrontMatterWithDefaults, currentTime time.Time) bool {
	if fm.DTStart.IsZero() {
		return false
	}

	today := currentTime.Truncate(24 * time.Hour)
	endDate := fm.DTStart.Add(fm.Duration)

	// Check if today falls within the event's active window
	return (today.Equal(fm.DTStart) || today.After(fm.DTStart)) && today.Before(endDate)
}

// isOneTimeTaskActive wrapper for backward compatibility
func isOneTimeTaskActive(fm *FrontMatter) bool {
	if fm.DTStart == "" {
		return false
	}

	today := time.Now().Truncate(24 * time.Hour)
	startDate := parseStartDate(fm.DTStart)
	duration, err := ParseDuration(fm.Duration)
	if err != nil {
		return false
	}

	endDate := startDate.Add(duration)

	// Check if today falls within the event's active window
	return (today.Equal(startDate) || today.After(startDate)) && today.Before(endDate)
}

// ParseStartDate parses dtstart string with fallback
func ParseStartDate(dtStartStr string, fallbackDate time.Time) time.Time {
	if dtStartStr == "" {
		return fallbackDate
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

	// If parsing fails, use fallback
	return fallbackDate
}

// parseStartDate wrapper for backward compatibility
func parseStartDate(dtStartStr string) time.Time {
	fallback := time.Now().AddDate(-1, 0, 0).Truncate(24 * time.Hour)
	return ParseStartDate(dtStartStr, fallback)
}

// ApplyDefaults applies default values to frontmatter
func ApplyDefaults(fm *FrontMatter, currentTime time.Time) (*FrontMatterWithDefaults, error) {
	duration, err := ParseDuration(fm.Duration)
	if err != nil {
		return nil, fmt.Errorf("duration parsing error: %w", err)
	}

	fallbackStartDate := currentTime.AddDate(-1, 0, 0).Truncate(24 * time.Hour)
	startDate := ParseStartDate(fm.DTStart, fallbackStartDate)

	return &FrontMatterWithDefaults{
		RRule:    fm.RRule,
		Duration: duration,
		DTStart:  startDate,
		Tags:     fm.Tags,
	}, nil
}

func processFile(path string) Task {
	fm, err := parseFrontMatter(path)
	if err != nil {
		if !strings.Contains(err.Error(), "no frontmatter") {
			fmt.Println("Error processing", path+":", err)
		}
		return Task{}
	}

	filename := cleanFilename(filepath.Base(path))

	if fm.RRule != "" {
		nextStart := getNextOccurrence(fm)
		dueDate := getCurrentDueDate(fm)
		return Task{Name: filename, RRule: fm.RRule, Duration: fm.Duration, NextStart: nextStart, DueDate: dueDate, FilePath: path}
	} else if fm.DTStart != "" {
		// Handle one-time events
		dueDate := getOneTimeDueDate(fm)
		startDate := parseStartDate(fm.DTStart)
		return Task{Name: filename, RRule: "ONCE", Duration: fm.Duration, NextStart: &startDate, DueDate: dueDate, FilePath: path}
	}
	return Task{}
}

// IsTaskActive checks if task is active at given time
func IsTaskActive(fm *FrontMatterWithDefaults, currentTime time.Time) (bool, error) {
	today := currentTime.Truncate(24 * time.Hour)

	if fm.RRule != "" {
		// Create RRULE with proper DTSTART
		rruleStr := "DTSTART:" + fm.DTStart.Format("20060102T000000Z") + "\nRRULE:" + fm.RRule
		r, err := rrule.StrToRRule(rruleStr)
		if err != nil {
			return false, fmt.Errorf("RRULE parsing error: %w", err)
		}

		// Get all occurrences from start date to today + duration
		// (we need to check a bit into the future in case an occurrence + duration overlaps with today)
		endDate := today.Add(fm.Duration)
		occurrences := r.Between(fm.DTStart, endDate, true)

		// Check if today falls within any occurrence's active window
		for _, occurrence := range occurrences {
			occurrenceStart := occurrence.Truncate(24 * time.Hour)
			occurrenceEnd := occurrenceStart.Add(fm.Duration)

			if (today.Equal(occurrenceStart) || today.After(occurrenceStart)) && today.Before(occurrenceEnd) {
				return true, nil
			}
		}

		return false, nil
	} else if !fm.DTStart.IsZero() {
		// Handle one-time events
		return IsOneTimeTaskActive(fm, currentTime), nil
	}

	return false, nil
}

// isTaskActive wrapper for backward compatibility (uses file I/O)
func isTaskActive(path string) (bool, error) {
	fm, err := parseFrontMatter(path)
	if err != nil {
		return false, nil // No front matter is not an error
	}

	fmWithDefaults, err := ApplyDefaults(fm, time.Now())
	if err != nil {
		return false, err
	}

	return IsTaskActive(fmWithDefaults, time.Now())
}

func cleanFilename(filename string) string {
	// Remove date prefixes like "2025-05-22 ", "2025-05-22_", "2025.05.22 ", etc.
	datePattern := regexp.MustCompile(`^(\d{4}[-_.]\d{1,2}[-_.]\d{1,2}[\s_-]*)+`)
	cleaned := datePattern.ReplaceAllString(filename, "")
	cleaned = strings.TrimSuffix(cleaned, ".md")

	return cleaned
}
