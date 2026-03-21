---
name: staging-deploy-on-merge
description: Deploy to staging environment when branches are merged
extensionPoints:
  - post-merge
appliesTo:
  - merge-branch:post-merge
prerequisites:
  - Staging environment configured on hosting platform
  - Deployment credentials in repository secrets
platform: cross-platform
---

### Staging Deploy on Merge

After a branch merge completes, deploy the updated code to the staging environment for pre-production validation.

```yaml
# .github/workflows/staging-deploy.yml
name: Deploy to Staging
on:
  push:
    branches: [main, develop]

jobs:
  deploy-staging:
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to Staging
        run: |
          # Platform-specific deploy command
          # Vercel: vercel --prod --scope staging
          # Railway: railway up --environment staging
          # Render: curl deploy hook
          # DO: doctl apps create-deployment $APP_ID
          echo "Deploy to staging environment"
```

**Post-deploy verification:**
- Run smoke tests against staging URL
- Verify critical paths (auth, data, integrations)
- Check monitoring dashboards for error spikes
