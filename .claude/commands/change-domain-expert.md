---
version: "v0.91.0"
description: Change domain specialist for this project
argument-hint: "[specialist-name] (optional)"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /change-domain-expert
Change the active domain specialist for this project.
---
## Prerequisites
- Framework v0.17.0+ installed
- `framework-config.json` exists
---
## Available Base Experts
| # | Specialist | Focus Area |
|---|------------|------------|
| 1 | Full-Stack-Developer | End-to-end web |
| 2 | Backend-Specialist | Server-side/APIs |
| 3 | Frontend-Specialist | UI/UX client-side |
| 4 | Mobile-Specialist | iOS/Android/cross-platform |
| 5 | Desktop-Application-Developer | Native desktop |
| 6 | Embedded-Systems-Engineer | Hardware-software |
| 7 | Game-Developer | Game engines/graphics |
| 8 | ML-Engineer | ML/AI systems |
| 9 | Data-Engineer | Pipelines/warehousing |
| 10 | Cloud-Solutions-Architect | Cloud infrastructure |
| 11 | SRE-Specialist | Reliability/ops |
| 12 | Systems-Programmer-Specialist | Low-level systems |
---
## Workflow
### Step 1: Read Current Config
```javascript
const fwconfig = require('./.claude/scripts/shared/lib/framework-config.js');
const config = fwconfig.read(process.cwd());
const currentSpecialist = config.domainSpecialist;
const frameworkPath = config.frameworkPath;
```
**Note:** Writes **top-level** `domainSpecialist`, NOT nested `projectType.domainSpecialist` (fixed in #2292).
### Step 2: Select New Specialist
**If argument:** Use specified name.
**If none:** Present numbered list; ask user (1-12) or name.
### Step 3: Validate
Must be one of the 12 Base Experts. If invalid, report error and stop.
### Step 4: Update framework-config.json
Use `framework-config.js` helper (validates schema; invalid rejected at write).
```javascript
const fwconfig = require('./.claude/scripts/shared/lib/framework-config.js');
const config = fwconfig.read(process.cwd());
config.domainSpecialist = newSpecialist;
fwconfig.write(process.cwd(), config);
```
If `fwconfig.write` throws, report to user and stop — do NOT retry with raw `fs.writeFileSync`.
### Step 5: Update CLAUDE.md
Replace `**Domain Specialist:**` line with new value. Update On-Demand Documentation table row for domain specialist path.
### Step 6: Update .claude/rules/03-startup.md
Update three elements:
1. `**Domain Specialist:**` metadata line
2. Startup sequence path: `Read \`{frameworkPath}/System-Instructions/Domain/Base/{new-specialist}.md\``
3. "Active Role" confirmation message
### Step 7: Load New Specialist
```bash
cat "{frameworkPath}/System-Instructions/Domain/Base/{new-specialist}.md"
```
### Step 8: Report
```
Domain specialist changed successfully.

Previous: {old-specialist}
New: {new-specialist}

The new specialist profile has been loaded and is now active.
```
---
## Example Usage
```
/change-domain-expert
→ Prompt for selection

/change-domain-expert Backend-Specialist
→ Switch to Backend-Specialist

/change-domain-expert 2
→ Switch to #2 (Backend-Specialist)
```
---
**End of Change Domain Expert**
