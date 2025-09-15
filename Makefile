.PHONY: run build clean release-test

# Default target
all: build

# Run the application
run:
	go run main.go

# Build the binary
build:
	go build -o obsidian-tasks main.go

# Test goreleaser configuration
release-test:
	goreleaser release --snapshot --clean

# Clean build artifacts
clean:
	rm -f obsidian-tasks
	rm -rf dist/