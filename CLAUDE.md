# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go CLI tool called `obsidian-tasks` that scans Obsidian markdown files for recurring tasks defined with iCal RRULE + DURATION semantics in YAML front matter. It displays active and inactive tasks with smart date indicators including due dates and next start dates.

## Development Commands

### Building and Running
- `make build` - Build the binary to `obsidian-tasks`
- `make run` - Run the application directly with `go run main.go`
- `go build -o obsidian-tasks main.go` - Direct build command

### Testing and Release
- `make release-test` - Test goreleaser configuration with snapshot build
- `goreleaser release --snapshot --clean` - Direct goreleaser test command

### Cleanup
- `make clean` - Remove built binary and dist artifacts

## Configuration

The application requires a notes directory to be configured through:
1. `OBSIDIAN_NOTES_DIR` environment variable, or
2. Config file (`config.yaml` or `config.yml`) with `notes_dir` field in:
   - Current directory
   - `~/.config/obsidian-tasks/`

## Architecture

### Core Components
- **main.go** - Single-file application containing all logic
- **FrontMatter struct** - Handles YAML parsing for `rrule`, `duration`, `dtstart`, and `tags` fields
- **Task struct** - Represents task with name, rrule, duration, next start date, and due date
- **Config struct** - Manages notes directory configuration

### Key Functions
- `getNotesDir()` - Configuration resolution with fallback hierarchy
- `parseFrontMatter(path)` - Common YAML front matter parsing (eliminates duplication)
- `processFile(path)` - Creates Task struct with all metadata including dates
- `isTaskActive(path)` - Determines if task is active using RRULE + DURATION window logic
- `getNextOccurrence(fm)` - Calculates next start date for inactive tasks
- `getCurrentDueDate(fm)` - Calculates due date for currently active tasks
- `parseDuration(str)` - Parses ISO 8601 duration format (P1D, P1W, PT2H, etc.)
- `parseStartDate(str)` - Parses dtstart with fallback to 1 year ago
- `printTasks()` - Unified display with color-coded date indicators
- `cleanFilename(filename)` - Removes date prefixes and file extensions for display

### Task Logic (RRULE + DURATION)
1. **RRULE** generates recurring occurrence dates from dtstart
2. **DURATION** defines active window length for each occurrence
3. **Active Task**: Today falls within any occurrence's [start, start+duration) window
4. **Due Date**: Last day of current active window (start + duration - 1 day)
5. **Next Start**: First occurrence date after today
6. **Default**: dtstart = 1 year ago, duration = P1D (1 day)

### Dependencies
- `github.com/fatih/color` - Terminal color output with date highlighting
- `github.com/teambition/rrule-go` - RFC 5545 recurrence rule parsing
- `gopkg.in/yaml.v3` - YAML front matter parsing

### Display Features
- **Active Tasks**: Show due dates with red warning (⚠️) if due today, yellow arrow (→) otherwise
- **Inactive Tasks**: Show next start dates with cyan arrow (→)
- **Color Scheme**: Green (active names), red (urgent due), yellow (due dates), cyan (next start), gray (inactive)

### Cross-Platform Build
- Simplified goreleaser configuration
- Supports: linux/amd64, linux/arm64, windows/amd64
- Archives: tar.gz (Linux), zip (Windows)