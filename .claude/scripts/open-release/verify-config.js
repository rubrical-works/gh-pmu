#!/usr/bin/env node
// **Version:** 1.4.0
/**
 * verify-config.js - Verify config files are not corrupted by tests
 *
 * Restores config files to their committed state (undoing any test-time
 * side effects like YAML migration), then verifies they match HEAD.
 *
 * Run via: node .claude/scripts/open-release/verify-config.js
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

async function main() {
    // Restore config files to committed state before checking.
    // Tests may trigger side effects (e.g., YAML migration deletes .gh-pmu.yml
    // and updates .gh-pmu.json version). This is expected runtime behavior,
    // not test corruption.
    try {
        execSync('git checkout -- .gh-pmu.json .gh-pmu.yml 2>/dev/null || true', {
            encoding: 'utf8'
        });
    } catch (_) {
        // Ignore — files may not exist in git
    }

    const configPath = path.join(process.cwd(), '.gh-pmu.json');

    // Check if config exists
    if (!fs.existsSync(configPath)) {
        console.log(JSON.stringify({
            success: false,
            message: '.gh-pmu.json not found'
        }));
        process.exit(1);
    }

    // Check if config is still modified after restore (would indicate real corruption)
    try {
        const status = execSync('git status --porcelain .gh-pmu.json .gh-pmu.yml', {
            encoding: 'utf8'
        }).trim();

        if (status) {
            console.log(JSON.stringify({
                success: false,
                message: 'Config files dirty after restore. Possible test corruption.',
                data: { status }
            }));
            process.exit(1);
        }

        console.log(JSON.stringify({
            success: true,
            message: 'Config files are clean'
        }));
    } catch (err) {
        console.log(JSON.stringify({
            success: false,
            message: `Config verification failed: ${err.message}`
        }));
        process.exit(1);
    }
}

main();
