# Obsidian Tasks

A CLI tool for managing recurring tasks in Obsidian using iCal RRULE and DURATION semantics. Scans your Obsidian vault for markdown files with recurring task definitions and displays active/inactive tasks with smart date indicators.

## Features

- 📅 **iCal RRULE Support** - Full RFC 5545 recurrence rule compatibility
- ⏱️ **Duration Windows** - Define how long tasks remain active using ISO 8601 durations
- 🎯 **Smart Display** - Shows active tasks with due dates and inactive tasks with next start dates
- 🚨 **Due Date Alerts** - Red highlighting for tasks due today
- 🔄 **Flexible Scheduling** - Monthly, weekly, daily patterns with custom intervals
- 📂 **Vault Integration** - Seamlessly works with your existing Obsidian notes

## Installation

### Using eget (Recommended)
```bash
eget harnyk/obsidian-tasks
```

### Using Go
```bash
go install github.com/harnyk/obsidian-tasks@latest
```

### Manual Download
Download binaries from [GitHub Releases](https://github.com/harnyk/obsidian-tasks/releases) for:
- Linux (amd64, arm64)
- Windows (amd64)

## Configuration

Set your Obsidian vault location using one of these methods:

### Environment Variable
```bash
export OBSIDIAN_NOTES_DIR="/path/to/your/obsidian/vault"
obsidian-tasks
```

### Config File
Create `config.yaml` in one of these locations:
- Current directory: `./config.yaml`
- User config: `~/.config/obsidian-tasks/config.yaml`

```yaml
notes_dir: "/path/to/your/obsidian/vault"
```

## Obsidian Note Format

Add recurring task metadata to your markdown files using YAML frontmatter:

### Basic Example
```markdown
---
tags:
  - rrule
rrule: FREQ=MONTHLY;BYMONTHDAY=1
duration: P3D
---

# Monthly Invoice Task

This task runs monthly on the 1st and stays active for 3 days.
```

### Required Fields

- **`rrule`** - RFC 5545 recurrence rule defining when the task starts
- **`duration`** - ISO 8601 duration defining how long the task stays active

### Optional Fields

- **`dtstart`** - Start date (defaults to 1 year ago if not specified)
- **`tags`** - Include `rrule` tag for easy filtering

## RRULE Examples

### Monthly Tasks
```yaml
# First of every month
rrule: FREQ=MONTHLY;BYMONTHDAY=1

# Last 5 days of every month
rrule: FREQ=MONTHLY;BYMONTHDAY=-5

# 15th of every month
rrule: FREQ=MONTHLY;BYMONTHDAY=15

# Every 3 months
rrule: FREQ=MONTHLY;INTERVAL=3;BYMONTHDAY=1
```

### Weekly Tasks
```yaml
# Every Monday
rrule: FREQ=WEEKLY;BYDAY=MO

# Every weekday
rrule: FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR

# Every 2 weeks on Monday
rrule: FREQ=WEEKLY;INTERVAL=2;BYDAY=MO
```

### Daily Tasks
```yaml
# Every day
rrule: FREQ=DAILY

# Every 3 days
rrule: FREQ=DAILY;INTERVAL=3

# Weekdays only
rrule: FREQ=DAILY;BYDAY=MO,TU,WE,TH,FR
```

## Duration Examples

```yaml
# Time periods
duration: P1D      # 1 day
duration: P3D      # 3 days
duration: P1W      # 1 week
duration: P1M      # 1 month
duration: P1Y      # 1 year

# Time components
duration: PT2H     # 2 hours
duration: PT30M    # 30 minutes
duration: PT1H30M  # 1 hour 30 minutes

# Combined
duration: P1DT2H   # 1 day 2 hours
```

## Usage Examples

### Financial Tasks
```markdown
---
tags: [rrule]
rrule: FREQ=MONTHLY;BYMONTHDAY=1
duration: P3D
---

# Invoice Generation
Generate and send monthly invoices on the 1st, deadline 3rd.
```

### Utility Payments
```markdown
---
tags: [rrule]
rrule: FREQ=MONTHLY;BYMONTHDAY=-5
duration: P5D
---

# Submit Meter Readings
Submit utility meter readings in the last 5 days of each month.
```

## Output Format

### Active Tasks
Shows tasks that are currently in their active window:
```
Active tasks:
  - Invoice Generation (FREQ=MONTHLY;BYMONTHDAY=1, P3D) → 2025-01-03
  - Morning Checklist (FREQ=DAILY, PT4H) ⚠️ 2025-01-15
```

- **Yellow arrow (→)** - Normal due date
- **Red warning (⚠️)** - Due today!

### Inactive Tasks
Shows tasks with their next activation date:
```
Inactive tasks:
  - Monthly Reports (FREQ=MONTHLY;BYMONTHDAY=15, P2D) → 2025-02-15
  - Weekly Review (FREQ=WEEKLY;BYDAY=MO, P1D) → 2025-01-20
```

- **Cyan arrow (→)** - Next start date

## Task Logic

1. **RRULE** generates recurring occurrence dates
2. **Duration** defines active window from each occurrence
3. **Active**: Today falls within any occurrence's active window
4. **Due Date**: Last day of current active window
5. **Next Start**: First occurrence after today

## Development

```bash
# Build
make build

# Run
make run

# Test release
make release-test

# Clean
make clean
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

WTFPL - see LICENSE file for details.