---
name: production-deploy-on-tag
description: Deploy to production when a release tag is pushed
extensionPoints:
  - post-tag
appliesTo:
  - prepare-release:post-tag
  - prepare-beta:post-tag
prerequisites:
  - Production environment configured on hosting platform
  - Deployment credentials in repository secrets
  - Tag naming convention (v* pattern)
platform: cross-platform
---

### Production Deploy on Tag

After a release tag is pushed, deploy the tagged version to the production environment.

```yaml
# .github/workflows/production-deploy.yml
name: Deploy to Production
on:
  push:
    tags: ['v*']

jobs:
  deploy-production:
    runs-on: ubuntu-latest
    environment: production
    steps:
      - uses: actions/checkout@v4
      - name: Extract Version
        id: version
        run: echo "version=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT
      - name: Deploy to Production
        run: |
          # Platform-specific production deploy
          echo "Deploying ${{ steps.version.outputs.version }} to production"
```

**Production safeguards:**
- Use GitHub environment protection rules (approvals, wait timers)
- Deploy with the exact commit referenced by the tag
- Monitor error rates and rollback if thresholds exceeded
