---
name: vercel-preview-deploy
description: Deploy preview environments on Vercel for each pull request
extensionPoints:
  - post-pr-create
appliesTo:
  - merge-branch:post-pr-create
  - prepare-release:post-pr-create
prerequisites:
  - Vercel account with project linked
  - VERCEL_TOKEN, VERCEL_ORG_ID, VERCEL_PROJECT_ID in repository secrets
platform: vercel
---

### Vercel Preview Deploy

After PR creation, trigger a Vercel preview deployment so reviewers can test changes in an isolated environment.

```yaml
# .github/workflows/vercel-preview.yml
name: Vercel Preview Deploy
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: amondnet/vercel-action@v25
        with:
          vercel-token: ${{ secrets.VERCEL_TOKEN }}
          vercel-org-id: ${{ secrets.VERCEL_ORG_ID }}
          vercel-project-id: ${{ secrets.VERCEL_PROJECT_ID }}
```

**After deployment:**
- Comment the preview URL on the PR
- Run smoke tests against the preview URL if configured
