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
	RRule string   `yaml:"rrule"`
	Tags  []string `yaml:"tags"`
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

	var activeTasks []string
	var inactiveTasks []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			if task := processFile(path); task != "" {
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

	if len(activeTasks) > 0 {
		color.New(color.FgYellow, color.Bold).Println("\nActive tasks:")
		for _, task := range activeTasks {
			color.New(color.FgGreen, color.Bold).Println(task)
		}
	}

	if len(inactiveTasks) > 0 {
		color.New(color.FgYellow, color.Bold).Println("\nInactive tasks:")
		for _, task := range inactiveTasks {
			color.New(color.FgHiBlack).Println(task)
		}
	}
}

func processFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("Read error:", path, err)
		return ""
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return "" // no frontmatter
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return ""
	}

	yamlPart := parts[1]
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(yamlPart), &fm); err != nil {
		fmt.Println("YAML parsing error:", path, err)
		return ""
	}

	if fm.RRule != "" {
		filename := cleanFilename(filepath.Base(path))
		return fmt.Sprintf("%s → %s", filename, fm.RRule)
	}
	return ""
}

func isTaskActive(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return false
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return false
	}

	yamlPart := parts[1]
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(yamlPart), &fm); err != nil {
		return false
	}

	if fm.RRule != "" {
		today := time.Now().Truncate(24 * time.Hour)
		// Set dtstart so the generator has a reference point
		r, err := rrule.StrToRRule("DTSTART:" + today.Format("20060102T000000Z") + "\nRRULE:" + fm.RRule)
		if err != nil {
			return false
		}

		// Check if today's date is in the list
		dates := r.Between(today, today.Add(24*time.Hour), true)
		for _, d := range dates {
			if d.Year() == today.Year() && d.YearDay() == today.YearDay() {
				return true
			}
		}
	}
	return false
}

func cleanFilename(filename string) string {
	// Remove date prefixes like "2025-05-22 ", "2025-05-22_", "2025.05.22 ", etc.
	datePattern := regexp.MustCompile(`^(\d{4}[-_.]\d{1,2}[-_.]\d{1,2}[\s_-]*)+`)
	cleaned := datePattern.ReplaceAllString(filename, "")

	// Remove .md extension if present
	if strings.HasSuffix(cleaned, ".md") {
		cleaned = strings.TrimSuffix(cleaned, ".md")
	}

	return cleaned
}
