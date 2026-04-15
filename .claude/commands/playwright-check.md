---
version: "v0.86.0"
description: Verify Playwright installation and browser availability
argument-hint: "[--fix]"
copyright: "Rubrical Works (c) 2026"
---

<!-- MANAGED -->
# /playwright-check

Verify Playwright is installed and browsers available. Report status with remediation steps.

## Usage

| Command | Description |
|---------|-------------|
| `/playwright-check` | Check installation and report issues |
| `/playwright-check --fix` | Attempt auto-fix for common issues |

## Verification Steps

### Step 0: Detect Project Context

```bash
if [ -f "CHARTER.md" ]; then
  PROJECT_NAME=$(grep -m1 "^# " CHARTER.md | sed 's/^# //')
fi
```

Use `$PROJECT_NAME` in output headers; fall back to generic.

### Step 1: Check package.json

```bash
test -f package.json || { echo "NO_PACKAGE_JSON"; exit 0; }
node -e "const pkg=require('./package.json'); const deps={...pkg.dependencies,...pkg.devDependencies}; console.log(deps['@playwright/test']||deps['playwright']||'NOT_FOUND')"
```

- **NO_PACKAGE_JSON:** "No package.json found. Playwright is not installed."
- **NOT_FOUND:** Suggest `npm install -D @playwright/test`

### Step 2: Check Playwright Version

```bash
npx playwright --version
```

Expected: version number (e.g., `Version 1.40.0`).

### Step 3: Check Browser Status

```bash
npx playwright install --dry-run 2>&1
```

Parse for installed browsers vs those needing download.

### Step 4: Verify Browser Launch (if Step 3 passes)

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

Header: `{ProjectName} - Playwright Installation Check` if charter detected, else generic.

### All Checks Pass

```
CodeForge - Playwright Installation Check
─────────────────────────────────────────
✓ Package installed: @playwright/test@1.40.0
✓ Chromium: installed
✓ Firefox: installed
✓ WebKit: installed
✓ Browser launch: success

All checks passed!
```

### Issues Found

```
✗ Chromium: NOT INSTALLED
✗ Firefox: NOT INSTALLED
✗ WebKit: NOT INSTALLED

Fix: Run `npx playwright install` to download browsers
```

### Package Not Found

```
✗ Package: NOT INSTALLED

Fix: Run `npm install -D @playwright/test`
```

### No package.json

```
✗ No package.json found

To add Playwright:
  npm init -y
  npm install -D @playwright/test
```

## Auto-Fix Mode (--fix)

| Issue | Auto-Fix Command |
|-------|------------------|
| Browsers not installed | `npx playwright install` |
| System deps missing (Linux) | `npx playwright install-deps` (may need sudo) |
| Corrupted install | `npx playwright install --force` |

**Note:** Package installation (`npm install`) is NOT auto-fixed to avoid modifying package.json without consent.

### Fix Flow

1. Run verification
2. If issues found and `--fix` specified: report "Attempting auto-fix...", run fix command, re-run verification, report final status

## Common Issues and Remediation

| Issue | Remediation |
|-------|-------------|
| Package not installed | `npm install -D @playwright/test` |
| Browsers not downloaded | `npx playwright install` |
| System dependencies (Linux) | `npx playwright install-deps` |
| Corrupted browsers | `npx playwright install --force` |
| Version mismatch | `npm update @playwright/test && npx playwright install` |

## Related

- **playwright-setup** skill - Detailed installation guide and CI patterns
- **electron-development** skill - Playwright with Electron apps

**End of /playwright-check Command**
