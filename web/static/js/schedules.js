// Global variable to store server time offset
let serverTimeOffset = 0;

// Get server's current time
function getServerTime() {
    return new Date(Date.now() + serverTimeOffset);
}

// Sync with server time
async function syncServerTime() {
    try {
        const response = await fetch('/api/server-time');
        const serverInfo = await response.json();

        // Calculate offset between server time and browser time
        const serverTime = new Date(serverInfo.current_time);
        const browserTime = new Date();
        serverTimeOffset = serverTime - browserTime;

        console.log('Server timezone:', serverInfo.timezone);
        console.log('Time offset (ms):', serverTimeOffset);
        console.log('Server time:', serverInfo.formatted_str);

        return serverInfo;
    } catch (error) {
        console.error('Error syncing server time:', error);
        return null;
    }
}

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

// Calculate next execution time for a schedule (using server time)
function calculateNextExecution(schedule) {
    const now = getServerTime(); // Use server time instead of browser time
    let nextExec = null;

    if (schedule.schedule_type === 'weekly' && schedule.day_of_week !== null) {
        // Calculate next occurrence of this day/time
        nextExec = new Date(now);
        const currentDay = now.getDay();
        const targetDay = schedule.day_of_week;

        // Days until target day (0 if today)
        let daysUntil = (targetDay - currentDay + 7) % 7;

        // Set the time
        nextExec.setHours(schedule.hour, schedule.minute, 0, 0);

        // If today but time has passed, schedule for next week
        if (daysUntil === 0 && now >= nextExec) {
            daysUntil = 7;
        }

        nextExec.setDate(now.getDate() + daysUntil);
    } else if (schedule.schedule_type === 'yearly' && schedule.month && schedule.day_of_month) {
        // Calculate next occurrence of this date/time
        nextExec = new Date(now.getFullYear(), schedule.month - 1, schedule.day_of_month, schedule.hour, schedule.minute, 0, 0);

        // If date has passed this year, use next year
        if (now >= nextExec) {
            nextExec.setFullYear(now.getFullYear() + 1);
        }
    } else if (schedule.schedule_type === 'once' && schedule.specific_date) {
        // One-time schedule
        nextExec = new Date(schedule.specific_date);
        nextExec.setHours(schedule.hour, schedule.minute, 0, 0);
    }

    return nextExec;
}

// Format countdown timer (using server time)
function formatCountdown(nextExec) {
    if (!nextExec) return 'N/A';

    const now = getServerTime(); // Use server time instead of browser time
    const diff = nextExec - now;

    if (diff <= 0) {
        return 'Due now!';
    }

    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));

    const parts = [];
    if (days > 0) parts.push(`${days}d`);
    if (hours > 0 || days > 0) parts.push(`${hours}h`);
    parts.push(`${minutes}m`);

    return parts.join(' ');
}

// Display schedules in the table
function displaySchedules(schedules) {
    const tbody = document.getElementById('schedules-tbody');
    tbody.innerHTML = '';

    if (schedules.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" style="text-align: center;">No schedules configured</td></tr>';
        return;
    }

    const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
    const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

    schedules.forEach(schedule => {
        const row = document.createElement('tr');
        const timeStr = `${schedule.hour.toString().padStart(2, '0')}:${schedule.minute.toString().padStart(2, '0')}`;

        let dayOrDateStr = '';
        if (schedule.schedule_type === 'weekly' && schedule.day_of_week !== null) {
            dayOrDateStr = dayNames[schedule.day_of_week];
        } else if (schedule.schedule_type === 'yearly' && schedule.month && schedule.day_of_month) {
            dayOrDateStr = `${monthNames[schedule.month - 1]} ${schedule.day_of_month}`;
        } else if (schedule.schedule_type === 'once' && schedule.specific_date) {
            const date = new Date(schedule.specific_date);
            dayOrDateStr = date.toLocaleDateString();
        }

        // Calculate countdown
        const nextExec = calculateNextExecution(schedule);
        const countdown = schedule.enabled ? formatCountdown(nextExec) : 'Disabled';
        const countdownStyle = countdown === 'Due now!' ? 'color: #d32f2f; font-weight: bold;' : 'font-family: monospace;';

        row.innerHTML = `
            <td>${escapeHtml(schedule.name)}</td>
            <td>${dayOrDateStr}</td>
            <td>${timeStr}</td>
            <td><span style="${countdownStyle}">${countdown}</span></td>
            <td><span class="status-badge ${schedule.enabled ? 'enabled' : 'disabled'}">
                ${schedule.enabled ? 'Enabled' : 'Disabled'}
            </span></td>
            <td>${schedule.last_run ? new Date(schedule.last_run).toLocaleString() : 'Never'}</td>
            <td>
                <button class="btn-edit" onclick="editSchedule('${schedule.id}')">✏️ Edit</button>
                <button class="btn-delete" onclick="deleteSchedule('${schedule.id}')">🗑️ Delete</button>
                <button class="btn-edit" onclick="viewScheduleLogs('${schedule.id}')" style="background: #17a2b8;">📊 Logs</button>
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
    document.getElementById('schedule-enabled').checked = true;
    document.getElementById('schedule-type').value = 'weekly';
    toggleScheduleType();
}

// Toggle between schedule types
function toggleScheduleType() {
    const scheduleType = document.getElementById('schedule-type').value;

    // Hide all options
    document.getElementById('weekly-options').style.display = 'none';
    document.getElementById('yearly-options').style.display = 'none';
    document.getElementById('yearly-day-options').style.display = 'none';
    document.getElementById('once-options').style.display = 'none';

    // Reset required attributes
    document.getElementById('schedule-day').required = false;
    document.getElementById('schedule-month').required = false;
    document.getElementById('schedule-day-of-month').required = false;
    document.getElementById('schedule-date').required = false;

    // Show relevant options based on type
    if (scheduleType === 'weekly') {
        document.getElementById('weekly-options').style.display = 'block';
        document.getElementById('schedule-day').required = true;
    } else if (scheduleType === 'yearly') {
        document.getElementById('yearly-options').style.display = 'block';
        document.getElementById('yearly-day-options').style.display = 'block';
        document.getElementById('schedule-month').required = true;
        document.getElementById('schedule-day-of-month').required = true;
    } else if (scheduleType === 'once') {
        document.getElementById('once-options').style.display = 'block';
        document.getElementById('schedule-date').required = true;
    }
}

// Save schedule (create or update)
async function saveSchedule(event) {
    event.preventDefault();

    const id = document.getElementById('schedule-id').value;
    const scheduleType = document.getElementById('schedule-type').value;

    const schedule = {
        name: document.getElementById('schedule-name').value,
        group_jid: document.getElementById('schedule-group').value,
        webhook_url: document.getElementById('schedule-webhook').value,
        schedule_type: scheduleType,
        hour: parseInt(document.getElementById('schedule-hour').value),
        minute: parseInt(document.getElementById('schedule-minute').value),
        enabled: document.getElementById('schedule-enabled').checked
    };

    if (scheduleType === 'weekly') {
        schedule.day_of_week = parseInt(document.getElementById('schedule-day').value);
    } else if (scheduleType === 'yearly') {
        schedule.month = parseInt(document.getElementById('schedule-month').value);
        schedule.day_of_month = parseInt(document.getElementById('schedule-day-of-month').value);
    } else if (scheduleType === 'once') {
        const dateValue = document.getElementById('schedule-date').value;
        if (dateValue) {
            schedule.specific_date = dateValue + 'T00:00:00Z';
        }
    }

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
        document.getElementById('schedule-hour').value = schedule.hour;
        document.getElementById('schedule-minute').value = schedule.minute;
        document.getElementById('schedule-enabled').checked = schedule.enabled;
        document.getElementById('schedule-type').value = schedule.schedule_type || 'weekly';

        // Handle different schedule types
        if (schedule.schedule_type === 'weekly' && schedule.day_of_week !== null) {
            document.getElementById('schedule-day').value = schedule.day_of_week;
        } else if (schedule.schedule_type === 'yearly' && schedule.month && schedule.day_of_month) {
            document.getElementById('schedule-month').value = schedule.month;
            document.getElementById('schedule-day-of-month').value = schedule.day_of_month;
        } else if (schedule.schedule_type === 'once' && schedule.specific_date) {
            const dateOnly = schedule.specific_date.split('T')[0];
            document.getElementById('schedule-date').value = dateOnly;
        }

        toggleScheduleType();

        document.getElementById('form-title').textContent = 'Edit Schedule';
        document.getElementById('schedule-form').style.display = 'block';
    } catch (error) {
        console.error('Error loading schedule:', error);
        showError('Failed to load schedule');
    }
}

// View logs for a specific schedule
function viewScheduleLogs(scheduleId) {
    window.location.href = `/execution-logs?schedule=${scheduleId}`;
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
    alert(message);
}

function showError(message) {
    alert(message);
}

// Update server time display (using server time)
function updateServerTime() {
    const now = getServerTime(); // Use server time instead of browser time
    const timeStr = now.toLocaleTimeString('en-US', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
    const dateStr = now.toLocaleDateString('en-US', {
        weekday: 'short',
        year: 'numeric',
        month: 'short',
        day: 'numeric'
    });
    document.getElementById('server-time').textContent = `${dateStr} ${timeStr}`;
}

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    // First, sync with server time
    await syncServerTime();

    // Then load schedules and groups
    loadSchedules();
    loadGroups();

    // Add event listener for schedule type toggle
    document.getElementById('schedule-type').addEventListener('change', toggleScheduleType);

    // Update server time every second
    updateServerTime();
    setInterval(updateServerTime, 1000);

    // Refresh countdown timers every minute
    setInterval(loadSchedules, 60000);

    // Re-sync server time every 5 minutes to account for drift
    setInterval(syncServerTime, 5 * 60 * 1000);
});
