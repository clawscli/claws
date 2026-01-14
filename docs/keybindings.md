# Key Bindings

Complete reference for all keyboard shortcuts in claws.

## General Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `h` / `l` | Navigate within category (service list) |
| `Enter` / `d` | View resource details |
| `Esc` | Go back |
| `q` / `Ctrl+c` | Quit |

## Views & Modes

| Key | Action |
|-----|--------|
| `:` | Command mode (e.g., `:ec2/instances`) |
| `:` + `Enter` | Go to services |
| `~` | Toggle Dashboard ↔ Services |
| `:pulse` | Go to dashboard |
| `:services` | Go to service browser |
| `/` | Filter mode (fuzzy search) |
| `A` | AI Chat (Bedrock) |
| `Ctrl+E` | Toggle compact header |
| `?` | Show help |

## Resource Browser

| Key | Action |
|-----|--------|
| `Tab` | Next resource type |
| `1-9` | Switch to resource type by number |
| `a` | Open actions menu |
| `m` | Mark resource for comparison |
| `d` | Describe (or diff if marked) |
| `c` | Clear filter and mark |
| `N` | Load next page (pagination) |
| `M` | Toggle inline metrics (EC2, RDS, Lambda) |
| `y` | Copy resource ID to clipboard |
| `Y` | Copy resource ARN to clipboard |
| `Ctrl+r` | Refresh (including metrics) |

## Profile & Region

| Key | Action |
|-----|--------|
| `R` | Select AWS region(s) (multi-select supported) |
| `P` | Select AWS profile(s) (multi-select supported) |

## Commands

| Command | Action |
|---------|--------|
| `:q` / `:quit` | Quit |
| `:login [name]` | AWS console login (default: `claws-login` profile) |
| `:ec2/instances` | Navigate to EC2 instances |
| `:sort <col>` | Sort by column (ascending) |
| `:sort desc <col>` | Sort by column (descending) |
| `:tag <filter>` | Filter by tag (e.g., `:tag Env=prod`) |
| `:tags` | Browse all tagged resources |
| `:diff <name>` | Compare current row with named resource |
| `:diff <n1> <n2>` | Compare two named resources |
| `:theme <name>` | Change color theme |
| `:autosave on/off` | Enable/disable config autosave |
| `:settings` | Show current settings |
| `:clear-history` | Clear navigation history (stack) |

## Mouse Support

| Action | Effect |
|--------|--------|
| Hover | Highlight item under cursor |
| Click | Select item / navigate |
| Scroll wheel | Scroll through lists |
| Click on tabs | Switch resource type |
| Back button | Navigate back (same as Esc) |

## Navigation Shortcuts (Context-dependent)

These shortcuts navigate to related resources based on the current context:

| Key | Action |
|-----|--------|
| `v` | View VPC / Versions |
| `s` | View Subnets / Streams / Stages |
| `g` | View Security Groups |
| `r` | View Route Tables / Roles / Resources |
| `e` | View Events / Executions / Endpoints |
| `l` | View CloudWatch Logs |
| `o` | View Outputs / Operations |
| `i` | View Images / Indexes |
| `D` | View Data Sources (AppSync) / Task Definitions (ECS) |

## Region Selector (`R` key)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Space` | Toggle region selection |
| `a` | Select all regions |
| `n` | Deselect all regions |
| `/` | Filter regions |
| `Enter` | Apply selection |
| `Esc` | Cancel |

Selected regions are queried in parallel; resources display with Region column.

## Profile Selector (`P` key)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Space` | Toggle profile selection |
| `l` | SSO login for selected profile |
| `L` | Console login for selected profile (`:login`) |
| `/` | Filter profiles |
| `Enter` | Apply selection |
| `Esc` | Cancel |

Selected profiles are queried in parallel; resources display with Profile and Account columns.
