---
version: "v0.94.0"
description: Change domain specialist for this project
argument-hint: "[specialist-name] (optional)"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /change-domain-expert
Change the active domain specialist and load it into the session.
**Prerequisites:** Framework v0.17.0+; `framework-config.json` in project root.
## Selection and Loading
`framework-manifest.json` `domainSpecialists` = every specialist that ships (selectable). `loadableSpecialists` = the subset safe to **inject**; outside it means **announce-only** — selectable and recorded as active role, content never loaded. Announce-only is the designed state for reference-catalog-format specialists; injecting them caused measured verbosity contamination (#2492).
Resolution and validation are **not** reimplemented here — this command and the startup hook both call `.claude/scripts/shared/lib/specialist-resolver.js`, so a mid-session selection gets the same allowlist and input validation as one read at startup.
**No documentation artifact to update.** The active role is *rendered* from config by `startup-hook.js`, not stored. Earlier versions rewrote a `**Domain Specialist:**` line in `CLAUDE.md` and `.claude/rules/03-startup.md`; neither line exists, so both steps were no-ops. Step 4 is the whole persistence mechanism — do NOT add steps editing prose files to match.
## Workflow
### Step 1: Read Current Configuration
```javascript
const fwconfig = require('./.claude/scripts/shared/lib/framework-config.js');
const config = fwconfig.read(process.cwd());
const currentSpecialist = config.domainSpecialist;
const frameworkPath = config.frameworkPath;
```
Writes the **top-level** `domainSpecialist`, NOT nested `projectType.domainSpecialist` — the nested key no consumer read (silent drift, fixed #2292).
### Step 2: Select New Specialist
Argument given → use it. Otherwise list selectable specialists from the manifest and **ASK USER** to choose by number or name, marking each not in `loadableSpecialists` as *(announce-only)* so the user knows it will not load content.
```javascript
const manifest = require('./framework-manifest.json');
const selectable = manifest.domainSpecialists;
const loadable = new Set(manifest.loadableSpecialists || []);
```
Do NOT hardcode a specialist list here — it drifts as specialists are added, dropped, or rewritten.
### Step 3: Resolve and Validate
```javascript
const { resolveSpecialist } = require('./.claude/scripts/shared/lib/specialist-resolver.js');
const result = resolveSpecialist({ cwd: process.cwd(), frameworkPath, domainSpecialist: newSpecialist });
```
| Status | Action |
|--------|--------|
| `loaded` | Continue to Step 4 |
| `announce-only` | Continue to Step 4; **nothing is injected** |
| `rejected` | Report `result.warning`, **STOP** — do not write config |
| `missing` | Report `result.warning`, **STOP** — do not write config |
The resolver checks name shape before touching the filesystem, then manifest membership, then the loadable gate. Do NOT pre-resolve a path yourself, and do NOT read a file the resolver declined — that reintroduces the arbitrary-file-disclosure surface #2503 closed.
### Step 4: Update framework-config.json
```javascript
const config = fwconfig.read(process.cwd());
config.domainSpecialist = newSpecialist;  // top-level, NOT nested under projectType
fwconfig.write(process.cwd(), config);
```
The helper validates against the schema before writing. If `fwconfig.write` throws, report it and stop — do not retry with raw `fs.writeFileSync`. This write is what makes the switch survive: config is re-read every session start, so the next session injects what this step recorded.
### Step 5: Activate the New Specialist
`loaded` → present `result.content` as the active role and deactivate the previous one:
```
Deactivate the {old-specialist} role. Its instructions no longer apply.
Adopt {new-specialist}, whose full instructions follow.
--- BEGIN DOMAIN SPECIALIST: {new-specialist} ---
{result.content}
--- END DOMAIN SPECIALIST: {new-specialist} ---
```
`announce-only` → deactivate the previous role, load nothing, and state that {new-specialist} is announce-only so no specialist instructions were loaded.
### Step 6: Report Completion
Report previous and new specialist, appending "(announce-only)" when applicable.
## Limitations
**Deactivation is an instruction, not a removal.** The previous specialist's text stays in context; the instruction directs it to be disregarded. True removal happens only at compaction.
**Repeated switches accumulate.** Each adds another specialist's content (~2.3K–3.7K tokens), persisting until compaction — switching repeatedly in one session costs materially more than switching once or restarting.
**Config-on-disk governs the next session.** In-session state is transient; what survives is the value Step 4 wrote. If Step 4 fails or is skipped, the next session reverts to the specialist on disk.
## Example Usage
```
/change-domain-expert                    → lists specialists (announce-only marked), prompts
/change-domain-expert Security-Engineer  → switches and loads its content
/change-domain-expert UX-Designer        → switches; announce-only, no content loaded
```
**End of Change Domain Expert**
