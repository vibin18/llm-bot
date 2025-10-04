// API Base URL
const API_BASE = '/api';

// Load status on page load
document.addEventListener('DOMContentLoaded', () => {
    loadStatus();
    loadGroups();
    loadWebhooks();
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

// Load webhooks
async function loadWebhooks() {
    const container = document.getElementById('webhooks-container');
    container.innerHTML = '<p class="loading">Loading webhooks...</p>';

    try {
        const response = await fetch(`${API_BASE}/webhooks`);
        const data = await response.json();
        const webhooks = data.webhooks || [];

        if (webhooks.length === 0) {
            container.innerHTML = '<p class="loading">No webhooks configured.</p>';
            return;
        }

        // Render webhooks
        container.innerHTML = '';
        webhooks.forEach(webhook => {
            const webhookDiv = document.createElement('div');
            webhookDiv.className = 'webhook-item';

            const infoDiv = document.createElement('div');
            infoDiv.className = 'webhook-info';

            const triggerDiv = document.createElement('div');
            triggerDiv.className = 'webhook-trigger';
            triggerDiv.textContent = webhook.sub_trigger;

            const urlDiv = document.createElement('div');
            urlDiv.className = 'webhook-url';
            urlDiv.textContent = webhook.url;

            infoDiv.appendChild(triggerDiv);
            infoDiv.appendChild(urlDiv);

            const deleteBtn = document.createElement('button');
            deleteBtn.className = 'delete-btn';
            deleteBtn.textContent = '🗑️ Delete';
            deleteBtn.onclick = () => deleteWebhook(webhook.sub_trigger);

            webhookDiv.appendChild(infoDiv);
            webhookDiv.appendChild(deleteBtn);

            container.appendChild(webhookDiv);
        });
    } catch (error) {
        console.error('Failed to load webhooks:', error);
        container.innerHTML = '<p class="error-message">Failed to load webhooks. Please try again.</p>';
    }
}

// Show add webhook form
function showAddWebhookForm() {
    document.getElementById('add-webhook-form').style.display = 'block';
    document.getElementById('add-webhook-btn').style.display = 'none';
}

// Hide add webhook form
function hideAddWebhookForm() {
    document.getElementById('add-webhook-form').style.display = 'none';
    document.getElementById('add-webhook-btn').style.display = 'inline-block';
    document.getElementById('sub-trigger').value = '';
    document.getElementById('webhook-url').value = '';
}

// Add webhook
async function addWebhook() {
    const subTrigger = document.getElementById('sub-trigger').value.trim();
    const url = document.getElementById('webhook-url').value.trim();

    if (!subTrigger || !url) {
        alert('Please fill in all fields');
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/webhooks`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                sub_trigger: subTrigger,
                url: url
            }),
        });

        const data = await response.json();

        if (response.ok) {
            showWebhookMessage('success', 'Webhook added successfully!');
            hideAddWebhookForm();
            loadWebhooks();
        } else {
            showWebhookMessage('error', data.message || 'Failed to add webhook.');
        }
    } catch (error) {
        console.error('Failed to add webhook:', error);
        showWebhookMessage('error', 'Failed to add webhook. Please try again.');
    }
}

// Delete webhook
async function deleteWebhook(subTrigger) {
    if (!confirm(`Delete webhook for "${subTrigger}"?`)) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/webhooks`, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                sub_trigger: subTrigger
            }),
        });

        const data = await response.json();

        if (response.ok) {
            showWebhookMessage('success', 'Webhook deleted successfully!');
            loadWebhooks();
        } else {
            showWebhookMessage('error', 'Failed to delete webhook.');
        }
    } catch (error) {
        console.error('Failed to delete webhook:', error);
        showWebhookMessage('error', 'Failed to delete webhook. Please try again.');
    }
}

// Show webhook message
function showWebhookMessage(type, text) {
    const container = document.querySelector('.webhooks-section');
    const messageDiv = document.createElement('div');
    messageDiv.className = `${type}-message`;
    messageDiv.textContent = text;
    messageDiv.style.display = 'block';

    container.insertBefore(messageDiv, container.firstChild);

    setTimeout(() => {
        messageDiv.remove();
    }, 3000);
}
