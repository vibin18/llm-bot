# WhatsApp Presence Tracking & Prometheus Metrics

This feature enables tracking the online/offline status of WhatsApp contacts and exposing the data as Prometheus metrics for monitoring and alerting.

## Features

- ✅ Real-time presence tracking (online/offline status)
- ✅ Automatic subscription to contact presence updates
- ✅ Prometheus metrics exposure
- ✅ Support for tracking group participants
- ✅ Last seen timestamp tracking
- ✅ Automatic cleanup of stale data (30+ days)
- ✅ Thread-safe concurrent access

## Prometheus Metrics

The following metrics are exposed at `/metrics`:

### `whatsapp_contact_online`
**Type:** Gauge
**Labels:** `jid`, `name`
**Description:** Current online status of WhatsApp contacts (1 = online, 0 = offline)

**Example:**
```
whatsapp_contact_online{jid="919876543210@s.whatsapp.net",name="John Doe"} 1
whatsapp_contact_online{jid="919123456789@s.whatsapp.net",name="Jane Smith"} 0
```

### `whatsapp_contact_status_changes_total`
**Type:** Counter
**Labels:** `jid`, `name`, `status`
**Description:** Total number of status changes for WhatsApp contacts

**Example:**
```
whatsapp_contact_status_changes_total{jid="919876543210@s.whatsapp.net",name="John Doe",status="online"} 45
whatsapp_contact_status_changes_total{jid="919876543210@s.whatsapp.net",name="John Doe",status="offline"} 42
```

###`whatsapp_contact_last_seen_timestamp_seconds`
**Type:** Gauge
**Labels:** `jid`, `name`
**Description:** Unix timestamp of when the contact was last seen online

**Example:**
```
whatsapp_contact_last_seen_timestamp_seconds{jid="919123456789@s.whatsapp.net",name="Jane Smith"} 1729715234
```

## Setup & Configuration

### 1. Enable Presence Tracking

Add the presence service to your main application:

```go
import (
    "github.com/vibin/whatsapp-llm-bot/internal/core/services"
)

// Create presence service
presenceService := services.NewPresenceService(logger)

// Start the service
err := presenceService.Start(ctx)
if err != nil {
    log.Fatal(err)
}

// Enable presence tracking in WhatsApp client
whatsappClient.EnablePresenceTracking()

// Register presence event handler
whatsappClient.OnPresence(func(event *domain.PresenceEvent) {
    presenceService.UpdatePresence(event)
})
```

### 2. Subscribe to Contacts

You can subscribe to presence updates in different ways:

#### Subscribe to a specific contact:
```go
err := whatsappClient.SubscribeToPresence("919876543210@s.whatsapp.net")
```

#### Subscribe to all participants in a group:
```go
err := whatsappClient.SubscribeToGroupParticipants("120363400902171371@g.us")
```

#### Subscribe to multiple groups:
```go
groups, _ := whatsappClient.GetGroups(ctx)
for _, group := range groups {
    if group.IsAllowed {
        whatsappClient.SubscribeToGroupParticipants(group.JID)
    }
}
```

### 3. Configure Prometheus Scraping

The metrics endpoint is available at:
```
http://localhost:8080/metrics
```

#### Prometheus Configuration (prometheus.yml):
```yaml
scrape_configs:
  - job_name: 'whatsapp-bot'
    scrape_interval: 30s
    static_configs:
      - targets: ['localhost:8080']
```

## API Endpoints

### Get All Tracked Presences
```bash
curl http://localhost:8080/api/presence
```

**Response:**
```json
[
  {
    "jid": "919876543210@s.whatsapp.net",
    "name": "John Doe",
    "is_online": true,
    "last_seen": "2025-10-23T21:30:00Z",
    "last_status_change": "2025-10-23T21:30:00Z"
  },
  {
    "jid": "919123456789@s.whatsapp.net",
    "name": "Jane Smith",
    "is_online": false,
    "last_seen": "2025-10-23T20:15:00Z",
    "last_status_change": "2025-10-23T20:45:00Z"
  }
]
```

### Get Specific Contact Presence
```bash
curl http://localhost:8080/api/presence/919876543210@s.whatsapp.net
```

### Get Online Count
```bash
curl http://localhost:8080/api/presence/stats
```

**Response:**
```json
{
  "total_contacts": 25,
  "online_count": 12,
  "offline_count": 13
}
```

## Monitoring & Alerting

### Example Prometheus Queries

**Number of currently online contacts:**
```promql
sum(whatsapp_contact_online)
```

**Contacts that recently went offline:**
```promql
whatsapp_contact_online == 0
```

**Rate of status changes (per hour):**
```promql
rate(whatsapp_contact_status_changes_total[1h])
```

**Contacts offline for more than 24 hours:**
```promql
(time() - whatsapp_contact_last_seen_timestamp_seconds) > 86400
```

### Example Alerting Rules

```yaml
groups:
  - name: whatsapp_presence
    rules:
      - alert: ContactOfflineForTooLong
        expr: (time() - whatsapp_contact_last_seen_timestamp_seconds{name="Important Contact"}) > 172800
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Contact {{ $labels.name }} offline for more than 48 hours"
          description: "{{ $labels.jid }} has been offline since {{ $value | humanizeDuration }}"

      - alert: AllContactsOffline
        expr: sum(whatsapp_contact_online) == 0
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "All tracked WhatsApp contacts are offline"
          description: "This might indicate a connection issue"
```

## Grafana Dashboard

### Sample Dashboard Panels

**1. Online Status Gauge:**
```json
{
  "targets": [{
    "expr": "sum(whatsapp_contact_online)",
    "legendFormat": "Online Contacts"
  }],
  "type": "stat"
}
```

**2. Status Changes Timeline:**
```json
{
  "targets": [{
    "expr": "rate(whatsapp_contact_status_changes_total[5m])",
    "legendFormat": "{{ name }} - {{ status }}"
  }],
  "type": "graph"
}
```

**3. Contact List with Status:**
```json
{
  "targets": [{
    "expr": "whatsapp_contact_online",
    "format": "table",
    "instant": true
  }],
  "type": "table",
  "transformations": [
    {
      "id": "organize",
      "options": {
        "excludeByName": {},
        "indexByName": {},
        "renameByName": {
          "name": "Contact",
          "Value": "Online"
        }
      }
    }
  ]
}
```

## Performance Considerations

- **Memory Usage:** Each tracked contact uses approximately 200 bytes
- **Cleanup:** Contacts inactive for 30+ days are automatically removed
- **Concurrency:** All operations are thread-safe with RWMutex
- **WhatsApp Limits:** WhatsApp may rate-limit presence subscriptions if you subscribe to too many contacts too quickly

## Troubleshooting

### Presence Updates Not Received

1. **Check if presence tracking is enabled:**
   ```go
   whatsappClient.EnablePresenceTracking()
   ```

2. **Verify subscription:**
   ```bash
   # Check logs for "Subscribed to presence for"
   docker logs whatsapp-llm-bot | grep "Subscribed to presence"
   ```

3. **WhatsApp Connection:**
   - Ensure WhatsApp is connected
   - Check network connectivity
   - Verify the contact allows presence sharing

### No Metrics Appearing

1. **Check Prometheus endpoint:**
   ```bash
   curl http://localhost:8080/metrics | grep whatsapp_contact
   ```

2. **Verify event handler registration:**
   ```bash
   # Should see presence event logs
   docker logs whatsapp-llm-bot | grep "Presence update"
   ```

## Privacy & Legal Considerations

⚠️ **Important:**
- Always respect user privacy and WhatsApp's Terms of Service
- Only track presence for legitimate business/monitoring purposes
- Inform users if you're tracking their online status
- Comply with GDPR and other privacy regulations
- Do not use this for stalking or harassment

## Limitations

- WhatsApp may not send presence updates for all contacts
- Some users may have disabled "Last Seen" in their privacy settings
- Presence updates are not guaranteed to be real-time
- Bulk subscriptions may be rate-limited by WhatsApp

## Example: Complete Integration

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "github.com/vibin/whatsapp-llm-bot/internal/adapters/primary/whatsapp"
    "github.com/vibin/whatsapp-llm-bot/internal/core/services"
)

func main() {
    ctx := context.Background()
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // Create WhatsApp client
    waClient, err := whatsapp.NewClient("./data/whatsapp_session", []string{}, logger)
    if err != nil {
        log.Fatal(err)
    }

    // Create and start presence service
    presenceService := services.NewPresenceService(logger)
    presenceService.Start(ctx)

    // Enable presence tracking
    waClient.EnablePresenceTracking()

    // Register presence handler
    waClient.OnPresence(func(event *domain.PresenceEvent) {
        presenceService.UpdatePresence(event)
    })

    // Start WhatsApp client
    err = waClient.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Wait for connection
    time.Sleep(5 * time.Second)

    // Subscribe to all allowed groups
    groups, _ := waClient.GetGroups(ctx)
    for _, group := range groups {
        if group.IsAllowed {
            logger.Info("Subscribing to group participants", "group", group.Name)
            waClient.SubscribeToGroupParticipants(group.JID)
        }
    }

    // Keep running
    select {}
}
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/vibin/whatsapp-llm-bot/issues
- Documentation: See `/docs` folder

---

**Built with ❤️ using whatsmeow and Prometheus**
