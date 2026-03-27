# Project Charter: GitHub Praxis Management Utility

**Status:** Active
**Last Updated:** 2026-03-27

## Vision

A GitHub CLI extension that streamlines project workflows by unifying issue tracking, sub-issue hierarchy, and workflow automation into a single cohesive tool.

## Current Focus

v1.4.x - Stability, security hardening, and documentation improvements

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.23 |
| Framework | Cobra CLI |
| API | GitHub GraphQL (go-gh, shurcooL-graphql) |

## In Scope (Current)

- Project field management (status, priority, custom fields)
- Sub-issue hierarchy with progress tracking
- Batch operations (intake, triage, split, batch mutations)
- Workflow automation (branch tracking)
- Terminal Kanban board visualization
- Cross-repository issue operations
- Auto-create labels and custom fields
- Label management (sync, add, update, delete)
- Config integrity verification
- Status transition validation
- E2E test infrastructure

---
*See Inception/ for full specifications*
