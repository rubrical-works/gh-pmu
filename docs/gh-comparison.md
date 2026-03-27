# gh vs gh pmu Command Comparison

This document compares `gh pmu` commands with the base GitHub CLI (`gh`) to help you understand when to use each.

## Command Overlap

| Function | `gh` Command | `gh pmu` Command | Notes |
|----------|--------------|------------------|-------|
| **Create issue** | `gh issue create` | `gh pmu create` | pmu adds project/status assignment in one step |
| **List issues** | `gh issue list` | `gh pmu list` | pmu filters by project board, shows project fields |
| **View issue** | `gh issue view` | `gh pmu view` | pmu shows project metadata (status, priority) |
| **Close issue** | `gh issue close` | `gh pmu close` | pmu adds reason aliases (wontfix, dupe, completed) |
| **Add to project** | `gh project item-add` | `gh pmu create` | pmu combines create + add + set fields |
| **Edit project field** | `gh project item-edit` | `gh pmu move` | Both update status/priority fields |
| **List project items** | `gh project item-list` | `gh pmu list` | Both list project items |

## Unique to gh pmu

### Unique Commands

These commands have no equivalent in the base `gh` CLI:

| Command | Purpose |
|---------|---------|
| `gh pmu board` | Terminal Kanban board view |
| `gh pmu comment` | Add comments to issues with stdin/file support |
| `gh pmu edit` | Edit issue body with round-trip file workflow |
| `gh pmu history` | Git commit history with issue reference parsing |
| `gh pmu init` | Configure project connection per-repo |
| `gh pmu intake` | Find issues not yet added to the project |
| `gh pmu move` | Update project fields with recursive sub-issue support |
| `gh pmu split` | Break an issue into sub-issues from checklist |
| `gh pmu sub` | Manage sub-issue hierarchy (add/create/list/remove) |
| `gh pmu triage` | Bulk rule-based issue processing |
| `gh pmu field` | Create and list project fields |
| `gh pmu branch` | Branch-based deployment workflow (releases, patches, hotfixes) |
| `gh pmu label` | Label management (sync, add, update, delete) |
| `gh pmu config` | Configuration integrity verification |
| `gh pmu validation` | Status transition validation rules |
| `gh pmu accept` | Terms and conditions acceptance gate |

### Unique Flags

Flags available in `gh pmu` that don't exist in base `gh`:

| Command | Flag | Purpose |
|---------|------|---------|
| `list` | `--status` | Filter by project status field |
| `list` | `--priority` | Filter by project priority field |
| `list` | `--has-sub-issues` | Show only parent issues |
| `create` | `--status` | Set project status on create |
| `create` | `--priority` | Set project priority on create |
| `create` | `--from-file` | Create issue from YAML/JSON file |
| `close` | `--update-status` | Move to 'done' status before closing |
| `move` | `--recursive` | Cascade changes to sub-issues |
| `move` | `--dry-run` | Preview changes without applying |
| `move` | `--depth` | Limit recursion depth |
| `move` | `--branch` | Assign to branch (use 'current' for active) |
| `move` | `--backlog` | Clear branch field |
| `move` | `--yes` | Skip confirmation prompt |
| `view` | `--body-file` | Export body to tmp/issue-{n}.md |
| `view` | `--body-stdout` | Output body to stdout |
| `view` | `--comments` | Show issue comments |
| `edit` | `--body-file` | Read body from file |
| `edit` | `--body-stdin` | Read body from stdin |
| `edit` | `--remove-label` | Remove labels from issue |
| `sub create` | `--inherit-labels` | Copy labels from parent |
| `sub create` | `--inherit-milestone` | Copy milestone from parent |
| `sub create` | `--inherit-assignees` | Copy assignees from parent |
| `split` | `--from` | Source: 'body' or file path |
| `split` | `--dry-run` | Preview what would be created |

## Unique to gh (use alongside gh pmu)

These base `gh` commands complement `gh pmu`:

| Command | Purpose |
|---------|---------|
| `gh issue develop` | Create linked branches |
| `gh issue pin/unpin` | Pin issues to repository |
| `gh issue transfer` | Move issue to another repository |
| `gh project field-*` | Manage project field definitions (pmu has `field create/list`) |
| `gh project copy/delete` | Project lifecycle management |

## When to Use Which

### Use `gh pmu` when:
- Working within a configured project board workflow
- You need project field values (status, priority) in output
- Managing sub-issue hierarchies
- Doing bulk operations (intake, triage)
- Wanting a visual board view in terminal

### Use base `gh` when:
- Working outside a project context
- Adding comments or editing issue content
- Managing project structure (fields, templates)
- Transferring issues between repos
- Creating branches from issues

## Common Workflows

### Update status and add comment
```bash
gh pmu move 123 --status in_progress
gh issue comment 123 --body "Starting work on this"
```

### Create issue with project tracking
```bash
# With gh pmu (one command)
gh pmu create --title "New feature" --status backlog --priority p1

# With base gh (multiple commands)
gh issue create --title "New feature"
gh project item-add --owner user --project 1 --url <issue-url>
gh project item-edit --project-id <id> --id <item-id> --field-id <status-id> --single-select-option-id <backlog-id>
```

### View issue with all context
```bash
# Project fields + issue details
gh pmu view 123

# Just issue details
gh issue view 123
```

## Summary

`gh pmu` is designed for **project-first workflows** where issues are tracked on a project board. It reduces multi-command sequences to single commands and adds features like sub-issue hierarchy and board visualization that don't exist in base `gh`.

Use `gh pmu` and `gh` together - they complement each other.
