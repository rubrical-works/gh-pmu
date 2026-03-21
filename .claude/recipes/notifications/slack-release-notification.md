---
name: slack-release-notification
description: Post to Slack when release is tagged
extensionPoints:
  - post-tag
  - post-close
appliesTo:
  - prepare-release:post-tag
  - prepare-release:post-close
  - prepare-beta:post-tag
prerequisites:
  - Slack incoming webhook URL
  - SLACK_WEBHOOK_URL environment variable
---

### Slack Notification

```bash
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"Release '$VERSION' has been tagged and deployed."}' \
  "$SLACK_WEBHOOK_URL"
```

**Note:** Set `SLACK_WEBHOOK_URL` environment variable or replace with your webhook.
