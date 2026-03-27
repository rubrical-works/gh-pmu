# Workflow Commands

gh-pmu provides branch workflow commands for managing development across feature releases, patches, and hotfixes.

## Branch

Branch workflows organize work into tracked branches with automatic issue management and artifact generation. The branch name is used literally for all artifacts.

### Starting a Branch

```bash
# Start a feature release
gh pmu branch start --name release/v2.0.0

# Start a patch release
gh pmu branch start --name patch/v1.9.1

# Start a hotfix
gh pmu branch start --name hotfix-auth-bypass
```

The command creates the git branch and a tracker issue with the `branch` label.

### Managing Issues

```bash
# Add issue to current branch
gh pmu branch add 42

# Remove issue from branch
gh pmu branch remove 42

# Create new issue directly in branch
gh pmu create --title "Add dark mode" --status in_progress --branch current
```

### Viewing Status

```bash
# Show current branch details
gh pmu branch current

# List all branches
gh pmu branch list

# Reopen a closed branch
gh pmu branch reopen release/v1.9.0
```

### Closing a Branch

```bash
# Close branch and generate artifacts
gh pmu branch close

# Close with git tag
gh pmu branch close --tag
```

**Generated artifacts** (configurable):
```
Releases/{branch}/
  release-notes.md    # Summary of included issues
  changelog.md        # Changes for this version
```

Examples:
- `Releases/release/v2.0.0/`
- `Releases/patch/v1.9.1/`
- `Releases/hotfix-auth-bypass/`

---

## Configuration

### Project Fields

Branch workflows require a `Branch` text field on the project. Run `gh pmu init` to auto-create it.

### Labels

Workflows use labels for tracker issues:

| Label | Used By |
|-------|---------|
| `branch` | Branch tracker issues |

Run `gh pmu init` to auto-create these labels.

### Branch Artifacts

Configure artifact generation in `.gh-pmu.json`:

```json
{
  "release": {
    "artifacts": {
      "directory": "Releases",
      "release_notes": true,
      "changelog": true
    }
  }
}
```

---

## Workflow Integration

### With Move Command

```bash
# Move issue and assign to current branch
gh pmu move 42 --status in_progress --branch current

# Clear branch assignment
gh pmu move 42 --backlog
```

### With Create Command

```bash
# Create issue in current branch
gh pmu create --title "New feature" --branch current
```

### Checking Active Branch

```bash
# See what branch is active
gh pmu branch current
```

---

## See Also

- [Commands Reference](commands.md) - Full command documentation
- [Configuration Guide](configuration.md) - `.gh-pmu.json` setup
- [Sub-Issues Guide](sub-issues.md) - Hierarchy management
