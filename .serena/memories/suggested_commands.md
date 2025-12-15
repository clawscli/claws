# Suggested Commands for claws Development

## Build & Run
```bash
# Using Task (recommended)
task build           # Build binary
task run             # Run application
task build && task run  # Build and run

# Using Go directly
go build ./cmd/claws
./claws

# Run with options
./claws -p myprofile          # Specific AWS profile
./claws -r us-west-2          # Specific region
./claws --read-only           # Read-only mode
./claws -l debug.log          # Enable debug logging
```

## Testing
```bash
# Using Task
task test            # Run all tests
task test-cover      # Run with coverage

# Using Go directly
go test ./...                    # Run all tests
go test -v ./...                 # Verbose output
go test ./internal/app/...       # Specific package
go test -race ./...              # With race detector
go test -cover ./internal/...    # With coverage
```

## Linting & Formatting
```bash
# Using Task
task lint            # Run linters
task fmt             # Format code

# Using Go directly
go fmt ./...
go vet ./...
golangci-lint run    # If installed
```

## Dependencies
```bash
go mod download      # Download dependencies
go mod tidy          # Tidy dependencies
go get -u ./...      # Update dependencies
```

## Debugging
```bash
# Run with debug logging
./claws -l claws.log

# Watch log output
tail -f claws.log

# Find dead code
go run golang.org/x/tools/cmd/deadcode@latest ./...
```

## Git Operations
```bash
git status
git add .
git commit -m "message"
git push
git log --oneline -10
```

## Code Analysis
```bash
# Count lines
find internal custom -name "*.go" -exec cat {} \; | wc -l

# Count files
find . -name "*.go" | wc -l

# Count resources
find custom -name "register.go" | wc -l

# Count services
ls custom/ | wc -l
```
