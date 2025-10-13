let webhooks = [];

// Load webhooks on page load
async function loadWebhooks() {
    try {
        const response = await fetch('/api/webhooks');
        const data = await response.json();
        webhooks = data.webhooks || [];
        displayWebhooks();
    } catch (error) {
        console.error('Error loading webhooks:', error);
        showError('Failed to load webhooks');
    }
}

// Display webhooks in table
function displayWebhooks() {
    const tbody = document.getElementById('webhooks-tbody');
    tbody.innerHTML = '';

    if (webhooks.length === 0) {
        tbody.innerHTML = '<tr><td colspan="3" style="text-align: center;">No webhooks configured</td></tr>';
        return;
    }

    webhooks.forEach(webhook => {
        const row = document.createElement('tr');
        const webhookUrl = webhook.webhook_url || webhook.url || '';
        const subTrigger = webhook.sub_trigger || '';

        row.innerHTML = `
            <td><code>${escapeHtml(webhookUrl)}</code></td>
            <td><strong>${escapeHtml(subTrigger)}</strong></td>
            <td>
                <button class="btn-delete" onclick="deleteWebhook('${escapeHtml(webhookUrl)}', '${escapeHtml(subTrigger)}')">🗑️ Delete</button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

// Show add webhook form
function showAddWebhookForm() {
    document.getElementById('webhook-form').style.display = 'block';
    document.getElementById('webhook-form-element').reset();
}

// Hide webhook form
function hideWebhookForm() {
    document.getElementById('webhook-form').style.display = 'none';
}

// Save webhook
async function saveWebhook(event) {
    event.preventDefault();

    const webhook = {
        url: document.getElementById('webhook-url').value,
        sub_trigger: document.getElementById('sub-trigger').value
    };

    try {
        const response = await fetch('/api/webhooks', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(webhook)
        });

        if (!response.ok) throw new Error('Failed to add webhook');

        hideWebhookForm();
        loadWebhooks();
        showSuccess('Webhook added successfully');
    } catch (error) {
        console.error('Error adding webhook:', error);
        showError('Failed to add webhook');
    }
}

// Delete webhook
async function deleteWebhook(webhookUrl, subTrigger) {
    if (!confirm('Are you sure you want to delete this webhook?')) return;

    try {
        const response = await fetch(`/api/webhooks?sub_trigger=${encodeURIComponent(subTrigger)}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete webhook');

        loadWebhooks();
        showSuccess('Webhook deleted successfully');
    } catch (error) {
        console.error('Error deleting webhook:', error);
        showError('Failed to delete webhook');
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
    loadWebhooks();
});
