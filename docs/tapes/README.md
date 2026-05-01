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

Use task commands from project root. These commands expect `vhs` to be installed on the host and use Docker only for the pinned LocalStack emulator:

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
- Runs host-installed VHS with the demo AWS config

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
AWS_CONFIG_FILE=scripts/demo-aws-config/config \
AWS_SHARED_CREDENTIALS_FILE=scripts/demo-aws-config/credentials \
vhs docs/tapes/demo.tape
```

## Notes

- `task demo:record` and `task test:vhs` run VHS on the host, so they work anywhere `vhs` and Docker are available
- Tapes use `localstack/localstack:4.12.0` for demo data (no real AWS credentials needed). Current LocalStack releases use account-based Hobby/commercial plans, so keep this pin unless you intentionally move to a token-backed LocalStack plan or another AWS emulator.
- Adjust `Sleep` durations if rendering is slow
