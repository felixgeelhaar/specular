# Slack Notifier Plugin for Specular

Send Specular notifications to Slack channels.

## Installation

```bash
# Build the plugin
cd examples/plugins/slack-notifier
go build -o slack-notifier .

# Install to plugins directory
mkdir -p ~/.specular/plugins/slack-notifier
cp slack-notifier plugin.yaml ~/.specular/plugins/slack-notifier/

# Verify installation
specular plugin list
specular plugin health slack-notifier
```

## Configuration

### 1. Create a Slack Incoming Webhook

1. Go to https://api.slack.com/apps
2. Create a new app or select an existing one
3. Enable "Incoming Webhooks"
4. Add a new webhook to your workspace
5. Copy the webhook URL

### 2. Configure the Plugin

Set the webhook URL in your Specular configuration:

```bash
# Via environment variable
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."

# Or configure in specular
specular plugin config slack-notifier webhook_url "$SLACK_WEBHOOK_URL"
```

## Usage

The plugin sends notifications for various Specular events:

### Event Types

| Event | Description |
|-------|-------------|
| `build.started` | Build execution started |
| `build.completed` | Build completed successfully |
| `build.failed` | Build failed with errors |
| `policy.violation` | Policy check detected violations |
| `auto.started` | Autonomous mode started |
| `auto.completed` | Autonomous mode finished |

### Message Format

Notifications include:
- Event title with status emoji
- Message body
- Additional data fields
- Timestamp and source

### Example Message

```
✅ Build Completed

Build 'feature-auth' completed successfully.

*Duration:* 2m 34s
*Files Changed:* 12
*Tests:* 156 passed

Specular | 2024-01-15T10:30:00Z
```

## Testing

Test the plugin locally:

```bash
# Test health check
echo '{"action":"health"}' | ./slack-notifier

# Test notification
echo '{
  "action": "notify",
  "event": "build.completed",
  "data": {
    "title": "Build Completed",
    "message": "Build completed successfully",
    "status": "success",
    "duration": "2m 34s"
  },
  "config": {
    "webhook_url": "https://hooks.slack.com/services/..."
  }
}' | ./slack-notifier
```

## Events Reference

### Build Events

```json
{
  "event": "build.completed",
  "data": {
    "title": "Build Completed",
    "message": "Build feature-auth completed",
    "status": "success",
    "duration": "2m 34s",
    "files_changed": 12
  }
}
```

### Policy Events

```json
{
  "event": "policy.violation",
  "data": {
    "title": "Policy Violation",
    "message": "Security policy check failed",
    "status": "error",
    "violations": 3,
    "severity": "high"
  }
}
```

### Autonomous Mode Events

```json
{
  "event": "auto.completed",
  "data": {
    "title": "Autonomous Mode Completed",
    "message": "Successfully implemented feature",
    "status": "success",
    "steps_completed": 5,
    "files_modified": 8
  }
}
```

## Troubleshooting

### Webhook Not Responding

1. Verify webhook URL is correct
2. Check Slack app has correct permissions
3. Ensure webhook is active in Slack settings

### Messages Not Formatted

1. Check JSON structure in data field
2. Verify required fields are present
3. Test locally with sample data

## License

MIT License - see LICENSE file for details.
