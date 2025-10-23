// Custom Dialog System

// Create dialog overlay and box
function createDialog(type, title, message) {
    const overlay = document.createElement('div');
    overlay.className = 'dialog-overlay';

    const iconMap = {
        success: '✓',
        error: '✕',
        warning: '⚠',
        info: 'ℹ'
    };

    overlay.innerHTML = `
        <div class="dialog-box">
            <div class="dialog-header">
                <div class="dialog-icon ${type}">${iconMap[type] || 'ℹ'}</div>
                <h3 class="dialog-title">${escapeHtml(title)}</h3>
            </div>
            <div class="dialog-content">
                ${escapeHtml(message)}
            </div>
        </div>
    `;

    return overlay;
}

// Show alert dialog
function showAlert(message, type = 'info', title = null) {
    return new Promise((resolve) => {
        const titleMap = {
            success: 'Success',
            error: 'Error',
            warning: 'Warning',
            info: 'Information'
        };

        const dialogTitle = title || titleMap[type] || 'Alert';
        const overlay = createDialog(type, dialogTitle, message);

        const footer = document.createElement('div');
        footer.className = 'dialog-footer';
        footer.innerHTML = `
            <button class="dialog-btn primary">OK</button>
        `;

        overlay.querySelector('.dialog-box').appendChild(footer);

        const handleClose = () => {
            overlay.style.opacity = '0';
            setTimeout(() => {
                document.body.removeChild(overlay);
                resolve();
            }, 200);
        };

        footer.querySelector('button').addEventListener('click', handleClose);
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) handleClose();
        });

        document.body.appendChild(overlay);
    });
}

// Show success message
function showSuccess(message, title = 'Success') {
    return showAlert(message, 'success', title);
}

// Show error message
function showError(message, title = 'Error') {
    return showAlert(message, 'error', title);
}

// Show warning message
function showWarning(message, title = 'Warning') {
    return showAlert(message, 'warning', title);
}

// Show info message
function showInfo(message, title = 'Information') {
    return showAlert(message, 'info', title);
}

// Show confirmation dialog
function showConfirm(message, title = 'Confirm', options = {}) {
    return new Promise((resolve) => {
        const {
            confirmText = 'Confirm',
            cancelText = 'Cancel',
            type = 'warning',
            danger = false
        } = options;

        const overlay = createDialog(type, title, message);

        const footer = document.createElement('div');
        footer.className = 'dialog-footer';

        const confirmButtonClass = danger ? 'danger' : 'primary';

        footer.innerHTML = `
            <button class="dialog-btn secondary" data-action="cancel">${escapeHtml(cancelText)}</button>
            <button class="dialog-btn ${confirmButtonClass}" data-action="confirm">${escapeHtml(confirmText)}</button>
        `;

        overlay.querySelector('.dialog-box').appendChild(footer);

        const handleClose = (result) => {
            overlay.style.opacity = '0';
            setTimeout(() => {
                document.body.removeChild(overlay);
                resolve(result);
            }, 200);
        };

        footer.addEventListener('click', (e) => {
            if (e.target.dataset.action === 'confirm') {
                handleClose(true);
            } else if (e.target.dataset.action === 'cancel') {
                handleClose(false);
            }
        });

        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) handleClose(false);
        });

        // Focus confirm button
        document.body.appendChild(overlay);
        setTimeout(() => {
            footer.querySelector('[data-action="confirm"]').focus();
        }, 100);
    });
}

// Utility function to escape HTML (if not already defined)
if (typeof escapeHtml === 'undefined') {
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}
