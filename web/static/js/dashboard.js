// Load dashboard statistics
async function loadDashboardStats() {
    try {
        // Load WhatsApp status
        const statusResponse = await fetch('/api/status');
        const status = await statusResponse.json();

        const statusBadge = document.getElementById('wa-status');
        if (status.is_authenticated) {
            statusBadge.textContent = '✓ Connected';
            statusBadge.className = 'status-badge connected';
            document.getElementById('qr-container').style.display = 'none';
        } else {
            statusBadge.textContent = '✗ Disconnected';
            statusBadge.className = 'status-badge disconnected';

            if (status.qr_code) {
                document.getElementById('qr-code').src = status.qr_code;
                document.getElementById('qr-container').style.display = 'block';
            }
        }

        // Load schedules count
        const schedulesResponse = await fetch('/api/schedules');
        const schedules = await schedulesResponse.json();
        const activeSchedules = schedules.filter(s => s.enabled).length;
        document.getElementById('active-schedules').textContent = activeSchedules;

        // Load execution stats
        let totalExecutions = 0;
        let successfulExecutions = 0;

        const promises = schedules.map(schedule =>
            fetch(`/api/schedules/${schedule.id}/executions?limit=100`)
                .then(res => res.json())
                .catch(() => [])
        );

        const results = await Promise.all(promises);
        const allExecutions = results.flat();

        totalExecutions = allExecutions.length;
        successfulExecutions = allExecutions.filter(e => e.success).length;

        document.getElementById('total-executions').textContent = totalExecutions;

        if (totalExecutions > 0) {
            const successRate = Math.round((successfulExecutions / totalExecutions) * 100);
            document.getElementById('success-rate').textContent = successRate + '%';
        } else {
            document.getElementById('success-rate').textContent = 'N/A';
        }

    } catch (error) {
        console.error('Error loading dashboard stats:', error);
    }
}

// Initialize dashboard
document.addEventListener('DOMContentLoaded', () => {
    loadDashboardStats();

    // Refresh stats every 30 seconds
    setInterval(loadDashboardStats, 30000);
});
