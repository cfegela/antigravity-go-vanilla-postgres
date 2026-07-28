/**
 * TaskFlow API Helper Client
 * Handles HTTP requests, credentials, and automatic token refresh retry logic.
 */
const API_BASE = '/api';

async function apiRequest(endpoint, options = {}) {
    const config = {
        headers: {
            'Content-Type': 'application/json',
            ...options.headers,
        },
        credentials: 'include', // Include HttpOnly cookies automatically
        ...options,
    };

    if (config.body && typeof config.body === 'object' && !(config.body instanceof FormData)) {
        config.body = JSON.stringify(config.body);
    }

    try {
        let response = await fetch(`${API_BASE}${endpoint}`, config);

        // Attempt silent token refresh on 401 Unauthorized (unless we are already attempting auth/login/register/refresh)
        if (response.status === 401 && !endpoint.startsWith('/auth/login') && !endpoint.startsWith('/auth/register') && !endpoint.startsWith('/auth/refresh')) {
            console.log('Access token expired. Attempting silent refresh...');
            const refreshed = await refreshToken();
            if (refreshed) {
                // Retry original request once
                response = await fetch(`${API_BASE}${endpoint}`, config);
            } else {
                // Refresh failed; handle logout
                handleSessionExpired();
                throw new Error('Session expired. Please log in again.');
            }
        }

        const data = await response.json().catch(() => ({}));

        if (!response.ok) {
            throw new Error(data.error || `HTTP ${response.status} Error`);
        }

        return data;
    } catch (err) {
        console.error(`API Error [${endpoint}]:`, err);
        throw err;
    }
}

async function refreshToken() {
    try {
        const res = await fetch(`${API_BASE}/auth/refresh`, {
            method: 'POST',
            credentials: 'include',
        });
        return res.ok;
    } catch {
        return false;
    }
}

function handleSessionExpired() {
    showToast('Session expired. Please sign in again.', 'error');
    if (typeof showAuthScreen === 'function') {
        showAuthScreen();
    }
}

function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.innerHTML = `
        <span>${escapeHtml(message)}</span>
        <button onclick="this.parentElement.remove()" style="background:none;border:none;color:inherit;cursor:pointer;font-size:1.1rem;">&times;</button>
    `;

    container.appendChild(toast);

    setTimeout(() => {
        if (toast.parentElement) {
            toast.style.opacity = '0';
            toast.style.transform = 'translateX(50px)';
            toast.style.transition = 'all 0.3s ease';
            setTimeout(() => toast.remove(), 300);
        }
    }, 4000);
}

function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}
