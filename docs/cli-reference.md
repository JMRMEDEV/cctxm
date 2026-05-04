# CCTXM — CLI Reference

## Global Flags

| Flag | Short | Description |
|---|---|---|
| `--session <id>` | `-s` | Use specific session (overrides active session) |
| `--project <name>` | `-p` | Target project from config mappings |
| `--verbose` | `-v` | Override filter mode to verbose |
| `--config <path>` | `-c` | Path to config file (default: `.cctxm/config.yaml`) |

## Commands

### `cctxm init`

Initialize cctxm in the current workspace.

- Creates `.cctxm/` directory structure
- Generates default `config.yaml`
- Backs up and replaces `.github/copilot/` with injected instructions
- Adds `.github/copilot/` to `.git/info/exclude`

```bash
cctxm init
cctxm init --skip-rules    # Don't touch .github/copilot/
```

### `cctxm exec [flags] -- <command>`

Execute a command with output filtering.

```bash
cctxm exec -- docker logs my-container
cctxm exec --project input-service -- npm test
cctxm exec --verbose -- npm run build
cctxm exec --strict -- docker-compose logs
```

Behavior:
1. Resolves working directory (from `--project` or current dir)
2. Runs command, captures full output to session raw log
3. Classifies command type (docker, test, build, generic)
4. Applies filter based on current mode (from task keywords)
5. Writes filtered output to session filtered log
6. Returns filtered output to stdout

### `cctxm read [flags] <file>`

Smart file reading.

```bash
cctxm read ./docs/api-spec.md           # Auto: full if small/md, filtered if large
cctxm read --full ./docs/api-spec.md     # Force full content
cctxm read --search "auth token" ./docs/api-spec.md  # Search specific terms
```

Behavior:
- `.md` files: always full content
- Files ≤ 5KB: full content
- Files > 5KB: extract sections matching task keywords
- `--full` overrides all heuristics
- `--search` overrides task keywords with explicit terms

### `cctxm task`

Manage task context for the active session.

```bash
cctxm task set "fix auth token refresh failing in input-service"
cctxm task show          # Show current task + detected keywords + filter mode
cctxm task clear         # Clear task context
cctxm task mode strict   # Manually override filter mode
cctxm task mode normal
cctxm task mode verbose
```

### `cctxm session`

Manage sessions.

```bash
cctxm session start "fix auth refresh"   # Create + activate new session
cctxm session list                        # List all sessions
cctxm session show                        # Show active session details
cctxm session restore <id>                # Restore and activate a session
cctxm session switch <id>                 # Alias for restore
cctxm session logs                        # List raw log files in active session
cctxm session logs <NNN>                  # Cat specific raw log
cctxm session clean                       # Remove sessions older than 7 days
cctxm session clean --all                 # Remove all sessions
```

### `cctxm rules`

Manage Copilot instruction injection.

```bash
cctxm rules inject       # (Re)generate .github/copilot/ from cctxm rules
cctxm rules restore      # Restore original .github/copilot/ from backup
cctxm rules show         # Show what would be injected
```

### `cctxm config`

View/edit configuration.

```bash
cctxm config show        # Print current config
cctxm config edit        # Open config in $EDITOR
cctxm config set projects.api ./services/api   # Set a config value
```

## Configuration File

Location: `.cctxm/config.yaml`

```yaml
# Project directory mappings
projects:
  input-service: ./services/input-service
  auth-service: ./services/auth-service
  ui: ./frontend
  org: ./project-org

# Default project (used when --project is omitted and cwd is workspace root)
default_project: ""

# Filter settings
filter:
  # Max lines returned for generic filter
  max_lines: 200
  # Context lines around matches in strict/normal mode
  context_lines: 10
  # File size threshold for smart reading (bytes)
  read_threshold: 5120
  # Extensions that always get full read
  full_read_extensions:
    - .md
    - .yaml
    - .yml
    - .toml
    - .json
    - .env.example

# Rules injection
rules:
  # Rule files to inject into Copilot instructions
  inject:
    - .cctxm/rules/*.md
  # Paths to suppress (backed up and replaced)
  suppress:
    - .github/copilot/

# Session settings
session:
  # Auto-clean sessions older than N days
  retention_days: 7
```
