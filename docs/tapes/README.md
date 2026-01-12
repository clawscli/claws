# VHS Tape Files

This directory contains [VHS](https://github.com/charmbracelet/vhs) tape files for generating demo GIFs and screenshots.

## Tape Files

| File | Purpose | Output |
|------|---------|--------|
| `demo.tape` | Main demo GIF | `docs/images/demo.gif` |
| `themes.tape` | Theme screenshots | `docs/images/theme-*.png` |
| `theme-light.tape` | Light theme (white bg) | `docs/images/theme-light.png` |
| `features.tape` | Feature screenshots | `docs/images/*.png` |
| `command-mode.tape` | Command suggestions/completion | `docs/images/cmd-*.png` |

## Usage (Recommended)

Use task commands from project root (requires Docker + Linux):

```bash
# Record everything (GIF + all screenshots)
task demo:record

# Record individual items
task demo:record:gif          # Main demo GIF only
task demo:record:themes       # Theme screenshots only
task demo:record:theme-light  # Light theme (requires white terminal bg)
task demo:record:features     # Feature screenshots only
task demo:record:command-mode # Command mode tests

# Run all tapes as integration tests
task test:vhs
```

This automatically:
- Builds the `claws` binary
- Starts LocalStack with demo data
- Runs VHS in Docker with proper environment

## Manual Usage

If you have VHS installed locally:

```bash
# Start LocalStack with demo data first
task localstack:demo-setup
task build

# Then run VHS from project root
AWS_ENDPOINT_URL=http://localhost:4566 \
AWS_ACCESS_KEY_ID=test \
AWS_SECRET_ACCESS_KEY=test \
vhs docs/tapes/demo.tape
```

## Notes

- `task demo:record` requires Linux (`--network host` for LocalStack access)
- Tapes use LocalStack for demo data (no real AWS credentials needed)
- Adjust `Sleep` durations if rendering is slow
