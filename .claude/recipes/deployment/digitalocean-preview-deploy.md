---
name: digitalocean-preview-deploy
description: Deploy preview environments on DigitalOcean App Platform for each pull request
extensionPoints:
  - post-pr-create
appliesTo:
  - merge-branch:post-pr-create
  - prepare-release:post-pr-create
prerequisites:
  - DigitalOcean account with App Platform app configured
  - DIGITALOCEAN_ACCESS_TOKEN in repository secrets
  - .do/app.yaml or app spec configured
platform: digitalocean
---

### DigitalOcean Preview Deploy

After PR creation, trigger a DigitalOcean App Platform deployment using the dev database or staging spec.

```yaml
# .github/workflows/do-preview.yml
name: DigitalOcean Preview Deploy
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install doctl
        uses: digitalocean/action-doctl@v2
        with:
          token: ${{ secrets.DIGITALOCEAN_ACCESS_TOKEN }}
      - name: Deploy to App Platform
        run: doctl apps create-deployment ${{ secrets.DO_APP_ID }}
```

**After deployment:**
- App Platform provides a deployment URL in the doctl output
- Comment the deployment URL on the PR
