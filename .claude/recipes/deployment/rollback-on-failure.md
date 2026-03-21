---
name: rollback-on-failure
description: Automatically rollback deployment when health checks or smoke tests fail
extensionPoints:
  - post-tag
appliesTo:
  - prepare-release:post-tag
  - prepare-beta:post-tag
prerequisites:
  - Previous stable deployment recorded (tag or deployment ID)
  - Rollback mechanism supported by hosting platform
platform: cross-platform
---

### Rollback on Failure

If post-deploy health checks or smoke tests fail, automatically rollback to the previous stable deployment.

```yaml
# Add to production deploy workflow after health check
- name: Rollback on Failure
  if: failure()
  run: |
    echo "Deployment failed — initiating rollback"
    # Platform-specific rollback:
    # Vercel: vercel rollback
    # Railway: railway rollback
    # Render: curl -X POST render API with previous deploy ID
    # DO: doctl apps create-deployment $APP_ID --revision $PREVIOUS_REVISION
    echo "Rollback complete"

- name: Notify on Rollback
  if: failure()
  run: |
    echo "::warning::Production deployment rolled back due to health check failure"
    # Optionally notify via Slack, email, or GitHub issue
```

**Rollback strategy:**
- Revert to the last known-good deployment, not the previous commit
- Preserve logs and artifacts from the failed deployment for debugging
- Create a GitHub issue automatically for investigation
