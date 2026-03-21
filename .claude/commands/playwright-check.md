---
version: "v0.67.1"
description: Verify Playwright installation and browser availability
argument-hint: "[--fix]"
copyright: "Rubrical Works (c) 2026"
---

<!-- MANAGED -->
# /playwright-check

Verify Playwright is properly installed and browsers are available. Reports status and provides remediation steps for any issues.

---

## Usage

| Command | Description |
|---------|-------------|
| `/playwright-check` | Check installation status and report issues |
| `/playwright-check --fix` | Attempt to fix common issues automatically |

---

## Execution Instructions

Run the verification steps below and report results in the specified format.

---

## Verification Steps

### Step 0: Detect Project Context

```bash
# Check for charter to get project name
if [ -f "CHARTER.md" ]; then
  PROJECT_NAME=$(grep -m1 "^# " CHARTER.md | sed 's/^# //')
fi
```

Use `$PROJECT_NAME` in output headers if set. Fall back to generic header if no charter.

### Step 1: Check package.json

```bash
# First check if package.json exists
test -f package.json || { echo "NO_PACKAGE_JSON"; exit 0; }

# Check if Playwright is in dependencies
node -e "const pkg = require('./package.json'); const deps = {...pkg.dependencies, ...pkg.devDependencies}; console.log(deps['@playwright/test'] || deps['playwright'] || 'NOT_FOUND')"
```

**If NO_PACKAGE_JSON:** Report gracefully: "No package.json found. Playwright is not installed in this project."

**If NOT_FOUND:** Report package not installed, suggest `npm install -D @playwright/test`

### Step 2: Check Playwright Version

```bash
npx playwright --version
```

**Expected:** Version number (e.g., `Version 1.40.0`)

### Step 3: Check Browser Status

```bash
# List installed browsers (dry-run shows what would be installed)
npx playwright install --dry-run 2>&1
```

**Parse output for:**
- Browsers already installed (shows path)
- Browsers needing download (shows "will download")

### Step 4: Verify Browser Launch (Optional - if Step 3 passes)

Create a temporary verification script:

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

---

## Output Format

**Header format:** If project name detected from CHARTER.md, use `{ProjectName} - Playwright Installation Check`. Otherwise use generic `Playwright Installation Check`.

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
CodeForge - Playwright Installation Check
─────────────────────────────────────────
✓ Package installed: @playwright/test@1.40.0
✗ Chromium: NOT INSTALLED
✗ Firefox: NOT INSTALLED
✗ WebKit: NOT INSTALLED

Issues Found:
1. Browsers not downloaded

Fix: Run `npx playwright install` to download browsers
```

### Package Not Found

```
CodeForge - Playwright Installation Check
─────────────────────────────────────────
✗ Package: NOT INSTALLED

Fix: Run `npm install -D @playwright/test` to install Playwright
```

### No package.json

```
CodeForge - Playwright Installation Check
─────────────────────────────────────────
✗ No package.json found

Playwright is not installed in this project.
To add Playwright, first initialize a Node.js project:
  npm init -y
  npm install -D @playwright/test
```

---

## Auto-Fix Mode (--fix)

When `--fix` is specified, attempt these fixes automatically:

| Issue | Auto-Fix Command |
|-------|------------------|
| Browsers not installed | `npx playwright install` |
| System deps missing (Linux) | `npx playwright install-deps` (may need sudo) |
| Corrupted install | `npx playwright install --force` |

**Note:** Package installation (`npm install`) is NOT auto-fixed to avoid modifying package.json without explicit consent.

### Fix Flow

1. Run verification
2. If issues found and `--fix` specified:
   - Report: "Attempting auto-fix..."
   - Run appropriate fix command
   - Re-run verification
   - Report final status

---

## Common Issues and Remediation

| Issue | Remediation |
|-------|-------------|
| Package not installed | `npm install -D @playwright/test` |
| Browsers not downloaded | `npx playwright install` |
| System dependencies (Linux) | `npx playwright install-deps` |
| Corrupted browsers | `npx playwright install --force` |
| Version mismatch | `npm update @playwright/test && npx playwright install` |

---

## Related

- **playwright-setup** skill - Detailed installation guide and CI patterns
- **electron-development** skill - Playwright with Electron apps

---

**End of /playwright-check Command**
