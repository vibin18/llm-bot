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

        row.innerHTML = `
            <td>${escapeHtml(schedule.name)}</td>
            <td>${dayOrDateStr}</td>
            <td>${timeStr}</td>
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

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadSchedules();
    loadGroups();

    // Add event listener for schedule type toggle
    document.getElementById('schedule-type').addEventListener('change', toggleScheduleType);
});
