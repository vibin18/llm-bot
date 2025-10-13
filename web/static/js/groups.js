let allGroups = [];
let allowedGroups = [];

// Load groups on page load
async function loadGroups() {
    try {
        // Load WhatsApp status
        const statusResponse = await fetch('/api/status');
        const status = await statusResponse.json();

        const statusBadge = document.getElementById('connection-status');
        if (status.is_authenticated) {
            statusBadge.textContent = '✓ Connected';
            statusBadge.className = 'status-badge allowed';
            document.getElementById('qr-container').style.display = 'none';
        } else {
            statusBadge.textContent = '✗ Disconnected';
            statusBadge.className = 'status-badge not-allowed';

            if (status.qr_code) {
                document.getElementById('qr-code').src = status.qr_code;
                document.getElementById('qr-container').style.display = 'block';
            }
        }

        // Load allowed groups
        const allowedResponse = await fetch('/api/config/allowed-groups');
        const allowedData = await allowedResponse.json();
        allowedGroups = allowedData.allowed_groups || [];

        // Load all groups
        const groupsResponse = await fetch('/api/groups');
        allGroups = await groupsResponse.json();

        displayGroups();
    } catch (error) {
        console.error('Error loading groups:', error);
        showError('Failed to load groups');
    }
}

// Display groups in table
function displayGroups() {
    const tbody = document.getElementById('groups-tbody');
    tbody.innerHTML = '';

    if (allGroups.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" style="text-align: center;">No groups found. Please connect WhatsApp first.</td></tr>';
        return;
    }

    allGroups.forEach(group => {
        const row = document.createElement('tr');
        const isAllowed = allowedGroups.includes(group.jid);

        row.innerHTML = `
            <td><strong>${escapeHtml(group.name)}</strong></td>
            <td><code>${escapeHtml(group.jid)}</code></td>
            <td>
                <span class="status-badge ${isAllowed ? 'allowed' : 'not-allowed'}">
                    ${isAllowed ? '✓ Allowed' : '✗ Not Allowed'}
                </span>
            </td>
            <td>
                ${isAllowed
                    ? `<button class="btn-remove" onclick="removeFromAllowed('${group.jid}')">Remove Access</button>`
                    : `<button class="btn-add" onclick="addToAllowed('${group.jid}')">Grant Access</button>`
                }
            </td>
        `;
        tbody.appendChild(row);
    });
}

// Add group to allowed list
async function addToAllowed(jid) {
    try {
        const updatedList = [...allowedGroups, jid];
        const response = await fetch('/api/config/allowed-groups', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ allowed_groups: updatedList })
        });

        if (!response.ok) throw new Error('Failed to update allowed groups');

        allowedGroups = updatedList;
        displayGroups();
        showSuccess('Group added to allowed list');
    } catch (error) {
        console.error('Error adding group:', error);
        showError('Failed to add group to allowed list');
    }
}

// Remove group from allowed list
async function removeFromAllowed(jid) {
    if (!confirm('Are you sure you want to remove this group from the allowed list?')) return;

    try {
        const updatedList = allowedGroups.filter(g => g !== jid);
        const response = await fetch('/api/config/allowed-groups', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ allowed_groups: updatedList })
        });

        if (!response.ok) throw new Error('Failed to update allowed groups');

        allowedGroups = updatedList;
        displayGroups();
        showSuccess('Group removed from allowed list');
    } catch (error) {
        console.error('Error removing group:', error);
        showError('Failed to remove group from allowed list');
    }
}

// Utility functions
function escapeHtml(text) {
    if (!text) return '';
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
    loadGroups();

    // Auto-refresh every 30 seconds
    setInterval(loadGroups, 30000);
});
