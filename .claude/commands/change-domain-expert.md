---
version: "v0.82.0"
description: Change domain specialist for this project
argument-hint: "[specialist-name] (optional)"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /change-domain-expert
Change the active domain specialist for this project.
## Prerequisites
- Framework v0.17.0+ installed
- `framework-config.json` exists in project root
## Available Base Experts
| # | Specialist | Focus Area |
|---|------------|------------|
| 1 | Full-Stack-Developer | End-to-end web development |
| 2 | Backend-Specialist | Server-side systems and APIs |
| 3 | Frontend-Specialist | UI/UX and client-side development |
| 4 | Mobile-Specialist | iOS, Android, cross-platform apps |
| 5 | Desktop-Application-Developer | Native desktop applications |
| 6 | Embedded-Systems-Engineer | Hardware-software integration |
| 7 | Game-Developer | Game engines and graphics |
| 8 | ML-Engineer | Machine learning and AI systems |
| 9 | Data-Engineer | Data pipelines and warehousing |
| 10 | Cloud-Solutions-Architect | Cloud infrastructure design |
| 11 | SRE-Specialist | Reliability and operations |
| 12 | Systems-Programmer-Specialist | Low-level systems programming |
## Workflow
### Step 1: Read Current Configuration
```bash
cat framework-config.json
```
Extract `frameworkPath` and current `projectType.domainSpecialist`.
### Step 2: Select New Specialist
**If argument provided:** Use the specified specialist name.
**If no argument:** Present numbered list, ask user to select (1-12) or type name.
### Step 3: Validate Selection
Must be one of the 12 Base Experts listed above. If invalid, report error and stop.
### Step 4: Update framework-config.json
Read file, update `projectType.domainSpecialist` to new value, write back.
### Step 5: Update CLAUDE.md
Find and replace the `**Domain Specialist:**` line and update On-Demand Documentation table row.
### Step 6: Update .claude/rules/03-startup.md
Update: 1) `**Domain Specialist:**` metadata line, 2) specialist file path, 3) "Active Role" message.
### Step 7: Load New Specialist
```bash
cat "{frameworkPath}/System-Instructions/Domain/Base/{new-specialist}.md"
```
### Step 8: Report Completion
```
Domain specialist changed successfully.
Previous: {old-specialist}
New: {new-specialist}
The new specialist profile has been loaded and is now active.
```
**End of Change Domain Expert**
