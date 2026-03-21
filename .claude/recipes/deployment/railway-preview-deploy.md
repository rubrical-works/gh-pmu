---
name: railway-preview-deploy
description: Deploy preview environments on Railway for each pull request
extensionPoints:
  - post-pr-create
appliesTo:
  - merge-branch:post-pr-create
  - prepare-release:post-pr-create
prerequisites:
  - Railway account with project linked
  - RAILWAY_TOKEN in repository secrets
platform: railway
---

### Railway Preview Deploy

After PR creation, trigger a Railway preview deployment using ephemeral environments.

```yaml
# .github/workflows/railway-preview.yml
name: Railway Preview Deploy
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install Railway CLI
        run: npm install -g @railway/cli
      - name: Deploy Preview
        run: railway up --environment pr-${{ github.event.pull_request.number }}
        env:
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
```

**After deployment:**
- Comment the preview URL on the PR
- Railway auto-cleans ephemeral environments when PR closes
