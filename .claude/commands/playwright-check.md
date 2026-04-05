---
version: "v0.82.0"
description: Verify Playwright installation and browser availability
argument-hint: "[--fix]"
copyright: "Rubrical Works (c) 2026"
---
<!-- MANAGED -->
# /playwright-check
Verify Playwright installation and browser availability. Reports status with remediation steps.

| Command | Description |
|---------|-------------|
| `/playwright-check` | Check installation status |
| `/playwright-check --fix` | Attempt to fix common issues |

## Verification Steps
### Step 0: Detect Project Context
```bash
if [ -f "CHARTER.md" ]; then
  PROJECT_NAME=$(grep -m1 "^# " CHARTER.md | sed 's/^# //')
fi
```
Use `$PROJECT_NAME` in output headers if set.
### Step 1: Check package.json
```bash
test -f package.json || { echo "NO_PACKAGE_JSON"; exit 0; }
node -e "const pkg = require('./package.json'); const deps = {...pkg.dependencies, ...pkg.devDependencies}; console.log(deps['@playwright/test'] || deps['playwright'] || 'NOT_FOUND')"
```
**NO_PACKAGE_JSON:** "No package.json found. Playwright is not installed in this project."
**NOT_FOUND:** Suggest `npm install -D @playwright/test`
### Step 2: Check Playwright Version
```bash
npx playwright --version
```
### Step 3: Check Browser Status
```bash
npx playwright install --dry-run 2>&1
```
Parse for installed browsers (path) vs needing download ("will download").
### Step 4: Verify Browser Launch (Optional - if Step 3 passes)
```javascript
// .playwright-verify.js
const { chromium } = require('playwright');
(async () => {
  try {
    const browser = await chromium.launch({ timeout: 10000 });
    console.log('LAUNCH_OK');
    await browser.close();
  } catch (e) {
    console.log('LAUNCH_FAILED: ' + e.message);
  }
})();
```
```bash
node .playwright-verify.js
rm .playwright-verify.js
```
## Output Format
**Header:** `{ProjectName} - Playwright Installation Check` if detected, else generic.
Report each check as pass/fail. On all pass: "All checks passed!" On issues: list failures with fix commands.
## Auto-Fix Mode (--fix)
| Issue | Auto-Fix Command |
|-------|------------------|
| Browsers not installed | `npx playwright install` |
| System deps missing (Linux) | `npx playwright install-deps` (may need sudo) |
| Corrupted install | `npx playwright install --force` |

Package installation (`npm install`) NOT auto-fixed to avoid modifying package.json without consent.
**Fix flow:** Verify -> if issues and `--fix`: attempt fix -> re-verify -> report.
## Common Issues
| Issue | Remediation |
|-------|-------------|
| Package not installed | `npm install -D @playwright/test` |
| Browsers not downloaded | `npx playwright install` |
| System dependencies (Linux) | `npx playwright install-deps` |
| Corrupted browsers | `npx playwright install --force` |
| Version mismatch | `npm update @playwright/test && npx playwright install` |
**End of /playwright-check Command**
