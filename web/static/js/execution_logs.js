let allExecutions = [];
let scheduleMap = {};

// Load execution logs
async function loadExecutionLogs() {
    try {
        // First load schedules to map IDs to names
        const schedulesResponse = await fetch('/api/schedules');
        const schedules = await schedulesResponse.json();

        schedules.forEach(schedule => {
            scheduleMap[schedule.id] = schedule.name;
        });

        // Populate schedule filter
        populateScheduleFilter(schedules);

        // Load all execution logs
        const promises = schedules.map(schedule =>
            fetch(`/api/schedules/${schedule.id}/executions?limit=50`)
                .then(res => res.json())
                .then(executions => executions.map(exec => ({
                    ...exec,
                    schedule_name: schedule.name
                })))
                .catch(() => [])
        );

        const results = await Promise.all(promises);
        allExecutions = results.flat().sort((a, b) =>
            new Date(b.executed_at) - new Date(a.executed_at)
        );

        updateStats();
        displayExecutions(allExecutions);
    } catch (error) {
        console.error('Error loading execution logs:', error);
        showError('Failed to load execution logs');
    }
}

// Populate schedule filter dropdown
function populateScheduleFilter(schedules) {
    const select = document.getElementById('schedule-filter');
    schedules.forEach(schedule => {
        const option = document.createElement('option');
        option.value = schedule.id;
        option.textContent = schedule.name;
        select.appendChild(option);
    });
}

// Update statistics
function updateStats() {
    const total = allExecutions.length;
    const successful = allExecutions.filter(e => e.success).length;
    const failed = total - successful;
    const successRate = total > 0 ? Math.round((successful / total) * 100) : 0;

    document.getElementById('total-executions').textContent = total;
    document.getElementById('success-count').textContent = successful;
    document.getElementById('failed-count').textContent = failed;
    document.getElementById('success-rate').textContent = successRate + '%';
}

// Display execution logs
function displayExecutions(executions) {
    const tbody = document.getElementById('logs-tbody');
    tbody.innerHTML = '';

    if (executions.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align: center;">No execution logs found</td></tr>';
        return;
    }

    executions.forEach(exec => {
        const row = document.createElement('tr');
        const executedDate = new Date(exec.executed_at);

        // Format response preview
        let responsePreview = '';
        if (exec.response) {
            responsePreview = exec.response.length > 100
                ? exec.response.substring(0, 100) + '...'
                : exec.response;
        }

        row.innerHTML = `
            <td><strong>${escapeHtml(exec.schedule_name || 'Unknown')}</strong></td>
            <td>${executedDate.toLocaleString()}</td>
            <td>
                <span class="status-badge ${exec.success ? 'success' : 'failed'}">
                    ${exec.success ? '✓ Success' : '✗ Failed'}
                </span>
            </td>
            <td>
                <div class="response-preview" title="${escapeHtml(exec.response || '')}">
                    ${escapeHtml(responsePreview)}
                </div>
            </td>
            <td>
                <div class="error-details" title="${escapeHtml(exec.error || '')}">
                    ${escapeHtml(exec.error || '-')}
                </div>
            </td>
        `;
        tbody.appendChild(row);
    });
}

// Filter executions
function filterExecutions() {
    const scheduleFilter = document.getElementById('schedule-filter').value;
    const statusFilter = document.getElementById('status-filter').value;

    let filtered = allExecutions;

    if (scheduleFilter) {
        filtered = filtered.filter(e => e.schedule_id === scheduleFilter);
    }

    if (statusFilter === 'success') {
        filtered = filtered.filter(e => e.success);
    } else if (statusFilter === 'failed') {
        filtered = filtered.filter(e => !e.success);
    }

    displayExecutions(filtered);
}

// Utility function
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showError(message) {
    alert(message);
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    // Check for schedule filter in URL
    const urlParams = new URLSearchParams(window.location.search);
    const scheduleId = urlParams.get('schedule');

    loadExecutionLogs().then(() => {
        if (scheduleId) {
            document.getElementById('schedule-filter').value = scheduleId;
            filterExecutions();
        }
    });

    // Add filter event listeners
    document.getElementById('schedule-filter').addEventListener('change', filterExecutions);
    document.getElementById('status-filter').addEventListener('change', filterExecutions);

    // Auto-refresh every 30 seconds
    setInterval(loadExecutionLogs, 30000);
});
