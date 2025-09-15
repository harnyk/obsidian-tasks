# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go CLI tool called `obsidian-tasks` that scans Obsidian markdown files for recurring tasks defined with RRULE (RFC 5545) in YAML front matter. It displays active and inactive tasks based on the current date and recurrence rules.

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
- **FrontMatter struct** - Handles YAML parsing for `rrule` and `tags` fields
- **Config struct** - Manages notes directory configuration

### Key Functions
- `getNotesDir()` - Configuration resolution with fallback hierarchy
- `processFile(path)` - Extracts and formats task information from markdown files
- `isTaskActive(path)` - Determines if a task is active today using RRULE evaluation
- `cleanFilename(filename)` - Removes date prefixes and file extensions for display

### Dependencies
- `github.com/fatih/color` - Terminal color output
- `github.com/teambition/rrule-go` - RFC 5545 recurrence rule parsing
- `gopkg.in/yaml.v3` - YAML front matter parsing

### File Processing Logic
1. Walks through all `.md` files in the configured notes directory
2. Parses YAML front matter for `rrule` field
3. Evaluates RRULE against current date to determine task status
4. Displays active tasks in green and inactive tasks in gray
5. Cleans filenames by removing date prefixes for better readability

### Cross-Platform Build
- Configured for Linux, Windows, macOS
- Supports multiple architectures (386, amd64, arm, arm64)
- Uses goreleaser for release automation