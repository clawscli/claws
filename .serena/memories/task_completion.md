# Task Completion Checklist for claws

When completing a task, ensure the following:

## Required Steps

### 1. Code Quality
- [ ] Code is formatted: `task fmt` or `go fmt ./...`
- [ ] No vet issues: `go vet ./...`
- [ ] No build errors: `task build` or `go build ./cmd/claws`
- [ ] No lint issues: `task lint`

### 2. Testing
- [ ] Tests pass: `task test` or `go test ./...`
- [ ] New code has appropriate tests (if applicable)
- [ ] Consider edge cases and error handling

### 3. Manual Verification
- [ ] Application runs: `./claws`
- [ ] New functionality works as expected
- [ ] No regressions in existing functionality

## Key Bindings to Test
| Key | Action |
|-----|--------|
| `j/k` | Navigate up/down |
| `Enter/d` | Describe resource |
| `:` | Command mode |
| `/` | Filter mode |
| `Tab` | Next resource type |
| `1-9` | Switch to resource type |
| `a` | Actions menu |
| `c` | Clear filter |
| `N` | Load next page (pagination) |
| `Ctrl+r` | Refresh |
| `R` | Switch region |
| `P` | Switch profile |
| `:sort <col>` | Sort by column |
| `?` | Help |
| `esc` | Back |
| `Ctrl+c` | Quit |

## Navigation Keys (Context-dependent)
| Key | Common Navigation |
|-----|-------------------|
| `v` | VPC |
| `s` | Subnets / Streams |
| `g` | Security Groups |
| `r` | Route Tables / Roles / Resources |
| `e` | Events / Executions |
| `l` | CloudWatch Logs |
| `o` | Outputs / Operations |
| `i` | Images / Indexes |

## Important Notes
- Mouse tracking is disabled (causes ESC key issues)
- Check both `custom/` and `internal/` directories for changes
- Update CLAUDE.md if significant changes are made
- Test with different AWS profiles/regions
- For new resources, test:
  - List view (columns display correctly)
  - Detail view (all fields show)
  - Navigation (if implemented)
  - Actions (if implemented)
