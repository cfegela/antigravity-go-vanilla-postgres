/**
 * TaskFlow Authentication Handler
 */

let currentUser = null;

function switchAuthTab(tab) {
    const loginTab = document.getElementById('tab-login');
    const registerTab = document.getElementById('tab-register');
    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form');

    if (tab === 'login') {
        loginTab.classList.add('active');
        loginTab.setAttribute('aria-selected', 'true');
        registerTab.classList.remove('active');
        registerTab.setAttribute('aria-selected', 'false');

        loginForm.classList.add('active');
        registerForm.classList.remove('active');
    } else {
        registerTab.classList.add('active');
        registerTab.setAttribute('aria-selected', 'true');
        loginTab.classList.remove('active');
        loginTab.setAttribute('aria-selected', 'false');

        registerForm.classList.add('active');
        loginForm.classList.remove('active');
    }
}

async function handleLogin(event) {
    event.preventDefault();
    const email = document.getElementById('login-email').value;
    const password = document.getElementById('login-password').value;
    const spinner = document.getElementById('login-spinner');
    const submitBtn = document.getElementById('login-submit-btn');

    try {
        spinner.classList.remove('hidden');
        submitBtn.disabled = true;

        const response = await apiRequest('/auth/login', {
            method: 'POST',
            body: { email, password },
        });

        currentUser = response.user;
        showToast(`Welcome back, ${currentUser.username}!`, 'success');
        showDashboard();
    } catch (err) {
        showToast(err.message || 'Login failed', 'error');
    } finally {
        spinner.classList.add('hidden');
        submitBtn.disabled = false;
    }
}

async function handleRegister(event) {
    event.preventDefault();
    const username = document.getElementById('register-username').value;
    const email = document.getElementById('register-email').value;
    const password = document.getElementById('register-password').value;
    const spinner = document.getElementById('register-spinner');
    const submitBtn = document.getElementById('register-submit-btn');

    try {
        spinner.classList.remove('hidden');
        submitBtn.disabled = true;

        const response = await apiRequest('/auth/register', {
            method: 'POST',
            body: { username, email, password },
        });

        currentUser = response.user;
        showToast(`Account created! Welcome, ${currentUser.username}!`, 'success');
        showDashboard();
    } catch (err) {
        showToast(err.message || 'Registration failed', 'error');
    } finally {
        spinner.classList.add('hidden');
        submitBtn.disabled = false;
    }
}

async function handleLogout() {
    try {
        await apiRequest('/auth/logout', { method: 'POST' });
        showToast('Logged out successfully', 'info');
    } catch {
        // Ignore logout errors
    } finally {
        currentUser = null;
        showAuthScreen();
    }
}

async function checkAuthSession() {
    try {
        const user = await apiRequest('/auth/me');
        currentUser = user;
        showDashboard();
    } catch {
        showAuthScreen();
    }
}

function showAuthScreen() {
    document.getElementById('auth-section').classList.remove('hidden');
    document.getElementById('app-dashboard').classList.add('hidden');
}

function showDashboard() {
    document.getElementById('auth-section').classList.add('hidden');
    document.getElementById('app-dashboard').classList.remove('hidden');

    if (currentUser) {
        document.getElementById('user-display-name').textContent = currentUser.username;
        document.getElementById('user-display-email').textContent = currentUser.email;
        document.getElementById('user-avatar').textContent = currentUser.username.charAt(0).toUpperCase();
    }

    // Load user's tasks
    if (typeof loadTasks === 'function') {
        loadTasks();
    }
}
