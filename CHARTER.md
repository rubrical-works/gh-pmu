# Project Charter: GitHub Praxis Management Utility

**Status:** Active
**Last Updated:** 2026-01-21

## Vision

A GitHub CLI extension that streamlines project workflows by unifying issue tracking, sub-issue hierarchy, and workflow automation into a single cohesive tool.

## Current Focus

v0.14.0 - Stability and documentation improvements

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.22 |
| Framework | Cobra CLI |
| API | GitHub GraphQL (go-gh, shurcooL-graphql) |

## In Scope (Current)

- Project field management (status, priority, custom fields)
- Sub-issue hierarchy with progress tracking
- Batch operations (intake, triage, split, batch mutations)
- Workflow automation (microsprint, branch tracking)
- Terminal Kanban board visualization
- Cross-repository issue operations
- Auto-create labels and custom fields
- E2E test infrastructure

---
*See Inception/ for full specifications*
