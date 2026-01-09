# devops-tui

A terminal user interface (TUI) for Azure DevOps Boards, inspired by
[jira-cli](https://github.com/ankitpokhrel/jira-cli) and
[JiraTUI](https://jiratui.sh/).

## Features

- View Azure DevOps work items in a clean terminal interface
- Filter by Sprint, State, and Assigned To
- Vim-style navigation (j/k/g/G)
- Fullscreen detail view
- Open work items in browser
- Cross-platform (Windows, macOS, Linux)
- **OAuth device flow authentication** - no need to manually create a
  PAT!

## Installation

### Build from source

```bash
go build -o devops-tui .
```

### Move to PATH

```bash
mv devops-tui /usr/local/bin/
```

## Authentication

### OAuth Device Flow (Recommended)

The easiest way to authenticate is to simply run the tool without
providing a PAT. It will automatically open your browser and guide you
through Microsoft's secure device login flow:

```bash
devops-tui
```

You'll see something like:

```
╭───────────────────────────────────────────────────────────────╮
│                  Azure DevOps Authentication                  │
├───────────────────────────────────────────────────────────────┤
│  To sign in, use a web browser to open:                       │
│  https://microsoft.com/devicelogin                            │
│                                                               │
│  And enter the code: ABC123DEF                                │
╰───────────────────────────────────────────────────────────────╯
```

The browser will open automatically. Sign in with your Azure DevOps
account, enter the code, and you're done! The token is cached securely
at `~/.config/devops-tui/token.json` and will be automatically
refreshed when needed.

### Personal Access Token (PAT)

If you prefer to use a PAT, you can still do so via environment
variable or config file:

```bash
export AZURE_DEVOPS_PAT="your-personal-access-token"
devops-tui
```

### Login

To explicitly authenticate or switch accounts:

```bash
devops-tui login
```

This will clear any cached credentials and start a fresh
authentication flow.

### Logout

To clear cached OAuth credentials:

```bash
devops-tui logout
```

## Configuration

Create a config file at `~/.config/devops-tui/config.yaml`:

```yaml
# Azure DevOps connection
organization: "my-organization"
project: "my-project"
team: "my-team"

# Authentication (optional - leave empty to use OAuth device flow)
# PAT can also be set via AZURE_DEVOPS_PAT environment variable
pat: ""

# UI settings
theme: "default"

# Default filters at startup
defaults:
  sprint: "current"
  state: "all"
  assigned: "me"
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `AZURE_DEVOPS_PAT` | Personal Access Token (optional - OAuth will be used if not set) |
| `AZURE_DEVOPS_ORG` | Organization (overrides config) |
| `AZURE_DEVOPS_PROJECT` | Project (overrides config) |
| `AZURE_DEVOPS_TEAM` | Team (overrides config) |

### PAT Permissions (if using PAT)

If you choose to use a Personal Access Token, it needs these scopes:
- `Work Items (Read)` - Read work items
- `Project and Team (Read)` - List sprints/iterations

## Keyboard Shortcuts

### Global

| Key | Description |
|-----|-------------|
| `Tab` | Switch to next panel |
| `Shift+Tab` | Switch to previous panel |
| `?` | Show/hide help |
| `Ctrl+r` | Reload data |
| `q` / `Ctrl+c` | Quit |

### Navigation

| Key | Description |
|-----|-------------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` | Go to first item |
| `G` | Go to last item |

### Actions

| Key | Description |
|-----|-------------|
| `Enter` / `Space` | Select filter / Open in browser |
| `v` | View fullscreen details |

### Detail View

| Key | Description |
|-----|-------------|
| `Esc` / `q` | Back to main view |
| `Enter` | Open in browser |
| `j` / `k` | Scroll description |

## Tech Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI
  framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - UI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Viper](https://github.com/spf13/viper) - Configuration

## License

MIT
