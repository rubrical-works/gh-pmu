---
name: render-preview-deploy
description: Deploy preview environments on Render for each pull request
extensionPoints:
  - post-pr-create
appliesTo:
  - merge-branch:post-pr-create
  - prepare-release:post-pr-create
prerequisites:
  - Render account with service configured
  - RENDER_API_KEY in repository secrets
  - render.yaml in repository root
platform: render
---

### Render Preview Deploy

After PR creation, trigger a Render preview deployment using pull request previews.

```yaml
# .github/workflows/render-preview.yml
name: Render Preview Deploy
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Trigger Render Deploy
        run: |
          curl -X POST "https://api.render.com/v1/services/$RENDER_SERVICE_ID/deploys" \
            -H "Authorization: Bearer ${{ secrets.RENDER_API_KEY }}" \
            -H "Content-Type: application/json" \
            -d '{"clearCache":"do_not_clear"}'
```

**After deployment:**
- Render provides automatic preview URLs for PRs when configured
- Check deploy status via the Render dashboard or API
