# Scheduler Integration Guide

## Remaining Steps

### 1. Update main.go to initialize scheduler

Add to imports:
```go
"github.com/vibin/whatsapp-llm-bot/internal/adapters/secondary/storage"
```

Add after webhook client initialization:
```go
// Initialize schedule repository
scheduleRepo, err := storage.NewScheduleRepository("data/schedules.db")
if err != nil {
    logger.Error("Failed to create schedule repository", "error", err)
    os.Exit(1)
}

// Initialize scheduler service
schedulerService := services.NewSchedulerService(scheduleRepo, webhookClient, whatsappClient, logger)

// Start scheduler
if err := schedulerService.Start(ctx); err != nil {
    logger.Error("Failed to start scheduler", "error", err)
}
defer schedulerService.Stop()
```

Update HTTP server initialization:
```go
scheduleHandlers := httpserver.NewScheduleHandlers(schedulerService)
server := httpserver.NewServer(cfg.App.Port, handlers, scheduleHandlers, logger)
```

### 2. Create UI Files

#### web/templates/schedules.html (NEW FILE)
See next section for full HTML code

#### web/static/js/schedules.js (NEW FILE)
```javascript
// Load schedules on page load
async function loadSchedules() {
    try {
        const response = await fetch('/api/schedules');
        const schedules = await response.json();
        displaySchedules(schedules || []);
    } catch (error) {
        console.error('Error loading schedules:', error);
        showError('Failed to load schedules');
    }
}

// Display schedules in the table
function displaySchedules(schedules) {
    const tbody = document.getElementById('schedules-tbody');
    tbody.innerHTML = '';

    if (schedules.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" style="text-align: center;">No schedules configured</td></tr>';
        return;
    }

    const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

    schedules.forEach(schedule => {
        const row = document.createElement('tr');
        const timeStr = `${schedule.hour.toString().padStart(2, '0')}:${schedule.minute.toString().padStart(2, '0')}`;

        row.innerHTML = `
            <td>${escapeHtml(schedule.name)}</td>
            <td>${dayNames[schedule.day_of_week]}</td>
            <td>${timeStr}</td>
            <td><span class="status-badge ${schedule.enabled ? 'enabled' : 'disabled'}">
                ${schedule.enabled ? 'Enabled' : 'Disabled'}
            </span></td>
            <td>${schedule.last_run ? new Date(schedule.last_run).toLocaleString() : 'Never'}</td>
            <td>
                <button class="btn-edit" onclick="editSchedule('${schedule.id}')">✏️ Edit</button>
                <button class="btn-delete" onclick="deleteSchedule('${schedule.id}')">🗑️ Delete</button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

// Show add schedule form
function showAddScheduleForm() {
    document.getElementById('schedule-form').style.display = 'block';
    document.getElementById('form-title').textContent = 'Add New Schedule';
    document.getElementById('schedule-id').value = '';
    resetForm();
}

// Hide schedule form
function hideScheduleForm() {
    document.getElementById('schedule-form').style.display = 'none';
    resetForm();
}

// Reset form
function resetForm() {
    document.getElementById('schedule-form-element').reset();
}

// Save schedule (create or update)
async function saveSchedule(event) {
    event.preventDefault();

    const id = document.getElementById('schedule-id').value;
    const schedule = {
        name: document.getElementById('schedule-name').value,
        group_jid: document.getElementById('schedule-group').value,
        webhook_url: document.getElementById('schedule-webhook').value,
        day_of_week: parseInt(document.getElementById('schedule-day').value),
        hour: parseInt(document.getElementById('schedule-hour').value),
        minute: parseInt(document.getElementById('schedule-minute').value),
        enabled: document.getElementById('schedule-enabled').checked
    };

    try {
        const url = id ? `/api/schedules/${id}` : '/api/schedules';
        const method = id ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(schedule)
        });

        if (!response.ok) throw new Error('Failed to save schedule');

        hideScheduleForm();
        loadSchedules();
        showSuccess(id ? 'Schedule updated successfully' : 'Schedule created successfully');
    } catch (error) {
        console.error('Error saving schedule:', error);
        showError('Failed to save schedule');
    }
}

// Edit schedule
async function editSchedule(id) {
    try {
        const response = await fetch(`/api/schedules/${id}`);
        const schedule = await response.json();

        document.getElementById('schedule-id').value = schedule.id;
        document.getElementById('schedule-name').value = schedule.name;
        document.getElementById('schedule-group').value = schedule.group_jid;
        document.getElementById('schedule-webhook').value = schedule.webhook_url;
        document.getElementById('schedule-day').value = schedule.day_of_week;
        document.getElementById('schedule-hour').value = schedule.hour;
        document.getElementById('schedule-minute').value = schedule.minute;
        document.getElementById('schedule-enabled').checked = schedule.enabled;

        document.getElementById('form-title').textContent = 'Edit Schedule';
        document.getElementById('schedule-form').style.display = 'block';
    } catch (error) {
        console.error('Error loading schedule:', error);
        showError('Failed to load schedule');
    }
}

// Delete schedule
async function deleteSchedule(id) {
    if (!confirm('Are you sure you want to delete this schedule?')) return;

    try {
        const response = await fetch(`/api/schedules/${id}`, { method: 'DELETE' });
        if (!response.ok) throw new Error('Failed to delete schedule');

        loadSchedules();
        showSuccess('Schedule deleted successfully');
    } catch (error) {
        console.error('Error deleting schedule:', error);
        showError('Failed to delete schedule');
    }
}

// Load groups for dropdown
async function loadGroups() {
    try {
        const response = await fetch('/api/groups');
        const groups = await response.json();
        const select = document.getElementById('schedule-group');

        select.innerHTML = '<option value="">Select a group...</option>';
        groups.forEach(group => {
            const option = document.createElement('option');
            option.value = group.jid;
            option.textContent = group.name;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Error loading groups:', error);
    }
}

// Utility functions
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showSuccess(message) {
    // TODO: Implement toast notification
    alert(message);
}

function showError(message) {
    // TODO: Implement toast notification
    alert(message);
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadSchedules();
    loadGroups();
});
```

### 3. Add Schedules Link to Admin UI

Update `web/templates/admin.html`:
Add after the webhooks section:
```html
<section class="schedules-section">
    <h2>📅 Schedules</h2>
    <div class="controls">
        <a href="/schedules" class="btn-primary">⏰ Manage Schedules</a>
    </div>
</section>
```

### 4. Create Schedules HTML Page (CRITICAL - COPY THIS)

Save as `web/templates/schedules.html`:
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Schedules - WhatsApp LLM Bot</title>
    <link rel="stylesheet" href="/static/css/style.css">
    <style>
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #667eea;
            color: white;
            font-weight: 600;
        }
        tr:hover {
            background-color: #f5f5f5;
        }
        .status-badge {
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.9rem;
        }
        .status-badge.enabled {
            background: #d4edda;
            color: #155724;
        }
        .status-badge.disabled {
            background: #f8d7da;
            color: #721c24;
        }
        .btn-edit, .btn-delete {
            padding: 6px 12px;
            margin: 0 4px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        .btn-edit {
            background: #4CAF50;
            color: white;
        }
        .btn-delete {
            background: #f44336;
            color: white;
        }
        .schedule-form {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            display: none;
        }
        .form-group {
            margin-bottom: 15px;
        }
        .form-group label {
            display: block;
            margin-bottom: 5px;
            font-weight: 500;
        }
        .form-group input, .form-group select {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 6px;
        }
        .form-actions {
            display: flex;
            gap: 10px;
            margin-top: 20px;
        }
        .btn-primary {
            background: #667eea;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 8px;
            cursor: pointer;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>📅 Schedules</h1>
            <p class="subtitle"><a href="/">← Back to Dashboard</a></p>
        </header>

        <div class="controls">
            <button class="btn-primary" onclick="showAddScheduleForm()">➕ Add Schedule</button>
        </div>

        <div id="schedule-form" class="schedule-form">
            <h3 id="form-title">Add New Schedule</h3>
            <form id="schedule-form-element" onsubmit="saveSchedule(event)">
                <input type="hidden" id="schedule-id">

                <div class="form-group">
                    <label for="schedule-name">Schedule Name*</label>
                    <input type="text" id="schedule-name" required>
                </div>

                <div class="form-group">
                    <label for="schedule-group">WhatsApp Group*</label>
                    <select id="schedule-group" required></select>
                </div>

                <div class="form-group">
                    <label for="schedule-webhook">Webhook URL*</label>
                    <input type="url" id="schedule-webhook" required>
                </div>

                <div class="form-group">
                    <label for="schedule-day">Day of Week*</label>
                    <select id="schedule-day" required>
                        <option value="0">Sunday</option>
                        <option value="1">Monday</option>
                        <option value="2">Tuesday</option>
                        <option value="3">Wednesday</option>
                        <option value="4">Thursday</option>
                        <option value="5">Friday</option>
                        <option value="6">Saturday</option>
                    </select>
                </div>

                <div class="form-group">
                    <label for="schedule-hour">Hour (0-23)*</label>
                    <input type="number" id="schedule-hour" min="0" max="23" required>
                </div>

                <div class="form-group">
                    <label for="schedule-minute">Minute (0-59)*</label>
                    <input type="number" id="schedule-minute" min="0" max="59" required>
                </div>

                <div class="form-group">
                    <label>
                        <input type="checkbox" id="schedule-enabled" checked>
                        Enabled
                    </label>
                </div>

                <div class="form-actions">
                    <button type="submit" class="btn-primary">Save</button>
                    <button type="button" onclick="hideScheduleForm()">Cancel</button>
                </div>
            </form>
        </div>

        <table>
            <thead>
                <tr>
                    <th>Name</th>
                    <th>Day</th>
                    <th>Time</th>
                    <th>Status</th>
                    <th>Last Run</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody id="schedules-tbody">
                <tr><td colspan="6" style="text-align: center;">Loading...</td></tr>
            </tbody>
        </table>
    </div>

    <script src="/static/js/schedules.js"></script>
</body>
</html>
```

### 5. Add Route to Serve Schedules Page

In `server.go`, add before the admin UI route:
```go
router.HandleFunc("/schedules", s.serveSchedulesUI).Methods("GET")
```

Add method to Server:
```go
func (s *Server) serveSchedulesUI(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "web/templates/schedules.html")
}
```

### 6. Build and Run

```bash
make docker-build
docker stop whatsapp-llm-bot && docker rm whatsapp-llm-bot
make docker-run
```

## Usage

1. Navigate to http://localhost:8080/schedules
2. Click "Add Schedule"
3. Fill in:
   - Name (e.g., "Daily News")
   - Select WhatsApp Group
   - Webhook URL
   - Day of week
   - Hour and minute
4. Save

The scheduler will trigger the webhook at the specified time each week and send the response to WhatsApp!
