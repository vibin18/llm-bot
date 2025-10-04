// API Base URL
const API_BASE = '/api';

// Load status on page load
document.addEventListener('DOMContentLoaded', () => {
    loadStatus();
    loadGroups();
    // Auto-refresh status every 5 seconds
    setInterval(loadStatus, 5000);
});

// Load WhatsApp connection status
async function loadStatus() {
    try {
        const response = await fetch(`${API_BASE}/status`);
        const data = await response.json();

        const statusBadge = document.getElementById('wa-status');
        const qrContainer = document.getElementById('qr-container');

        if (data.is_authenticated) {
            statusBadge.textContent = 'Connected';
            statusBadge.className = 'status-badge connected';
            qrContainer.style.display = 'none';
        } else {
            statusBadge.textContent = 'Disconnected';
            statusBadge.className = 'status-badge disconnected';

            if (data.qr_code) {
                qrContainer.style.display = 'block';
                document.getElementById('qr-code').src = data.qr_code;
            }
        }
    } catch (error) {
        console.error('Failed to load status:', error);
    }
}

// Load WhatsApp groups
async function loadGroups() {
    const container = document.getElementById('groups-container');
    container.innerHTML = '<p class="loading">Loading groups...</p>';

    try {
        // Load all groups
        const groupsResponse = await fetch(`${API_BASE}/groups`);
        const groups = await groupsResponse.json();

        // Load currently allowed groups
        const allowedResponse = await fetch(`${API_BASE}/config/allowed-groups`);
        const allowedData = await allowedResponse.json();
        const allowedSet = new Set(allowedData.allowed_groups || []);

        if (!groups || groups.length === 0) {
            container.innerHTML = '<p class="loading">No groups found. Make sure WhatsApp is connected.</p>';
            return;
        }

        // Render groups
        container.innerHTML = '';
        groups.forEach(group => {
            const groupDiv = document.createElement('div');
            groupDiv.className = 'group-item';

            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.className = 'group-checkbox';
            checkbox.checked = allowedSet.has(group.jid);
            checkbox.dataset.jid = group.jid;

            const infoDiv = document.createElement('div');
            infoDiv.className = 'group-info';

            const nameDiv = document.createElement('div');
            nameDiv.className = 'group-name';
            nameDiv.textContent = group.name || 'Unnamed Group';

            const jidDiv = document.createElement('div');
            jidDiv.className = 'group-jid';
            jidDiv.textContent = group.jid;

            infoDiv.appendChild(nameDiv);
            infoDiv.appendChild(jidDiv);

            const participantsSpan = document.createElement('span');
            participantsSpan.className = 'group-participants';
            participantsSpan.textContent = `${group.participants || 0} members`;

            groupDiv.appendChild(checkbox);
            groupDiv.appendChild(infoDiv);
            groupDiv.appendChild(participantsSpan);

            container.appendChild(groupDiv);
        });
    } catch (error) {
        console.error('Failed to load groups:', error);
        container.innerHTML = '<p class="error-message">Failed to load groups. Please try again.</p>';
    }
}

// Save allowed groups
async function saveAllowedGroups() {
    const checkboxes = document.querySelectorAll('.group-checkbox');
    const allowedGroups = [];

    checkboxes.forEach(checkbox => {
        if (checkbox.checked) {
            allowedGroups.push(checkbox.dataset.jid);
        }
    });

    try {
        const response = await fetch(`${API_BASE}/config/allowed-groups`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ groups: allowedGroups }),
        });

        const data = await response.json();

        if (response.ok) {
            showMessage('success', 'Allowed groups updated successfully!');
        } else {
            showMessage('error', 'Failed to update allowed groups.');
        }
    } catch (error) {
        console.error('Failed to save allowed groups:', error);
        showMessage('error', 'Failed to save changes. Please try again.');
    }
}

// Show success/error message
function showMessage(type, text) {
    const container = document.querySelector('.groups-section');
    const messageDiv = document.createElement('div');
    messageDiv.className = `${type}-message`;
    messageDiv.textContent = text;
    messageDiv.style.display = 'block';

    container.insertBefore(messageDiv, container.firstChild);

    setTimeout(() => {
        messageDiv.remove();
    }, 3000);
}
