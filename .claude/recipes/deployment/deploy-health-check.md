---
name: deploy-health-check
description: Verify deployment health after staging or production deploys
extensionPoints:
  - post-merge
  - post-tag
appliesTo:
  - merge-branch:post-merge
  - prepare-release:post-tag
  - prepare-beta:post-tag
prerequisites:
  - Health check endpoint configured (e.g., /api/health or /healthz)
  - Deployment URL available as environment variable or output
platform: cross-platform
---

### Post-Deploy Health Check

After a deployment completes, verify the application is healthy and responding correctly.

```yaml
# Add to existing deploy workflow as a post-deploy step
- name: Health Check
  run: |
    DEPLOY_URL="${{ steps.deploy.outputs.url }}"
    MAX_RETRIES=5
    RETRY_DELAY=10

    for i in $(seq 1 $MAX_RETRIES); do
      STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$DEPLOY_URL/api/health")
      if [ "$STATUS" = "200" ]; then
        echo "Health check passed (attempt $i)"
        exit 0
      fi
      echo "Health check failed (attempt $i, status: $STATUS), retrying in ${RETRY_DELAY}s..."
      sleep $RETRY_DELAY
    done
    echo "Health check failed after $MAX_RETRIES attempts"
    exit 1
```

**Health check targets:**
- HTTP status code (200 OK)
- Response time under threshold (e.g., < 2s)
- Database connectivity verified
- External service dependencies reachable
