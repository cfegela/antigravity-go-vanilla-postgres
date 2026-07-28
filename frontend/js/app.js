/**
 * TaskFlow CRUD Application Controller
 */

let allTasks = [];
let searchDebounceTimer = null;

// Initialize App
document.addEventListener('DOMContentLoaded', () => {
    checkAuthSession();
});

// Load tasks from API with active filters
async function loadTasks() {
    const status = document.getElementById('filter-status').value;
    const priority = document.getElementById('filter-priority').value;
    const search = document.getElementById('search-input').value.trim();

    let queryParams = new URLSearchParams();
    if (status && status !== 'all') queryParams.append('status', status);
    if (priority && priority !== 'all') queryParams.append('priority', priority);
    if (search) queryParams.append('q', search);

    const queryString = queryParams.toString() ? `?${queryParams.toString()}` : '';

    try {
        const tasks = await apiRequest(`/tasks${queryString}`);
        allTasks = tasks;
        renderTasks(tasks);
        updateStats(tasks);
    } catch (err) {
        showToast(err.message || 'Failed to load tasks', 'error');
    }
}

// Render Task Cards Grid
function renderTasks(tasks) {
    const grid = document.getElementById('tasks-grid');
    const emptyState = document.getElementById('empty-state');

    if (!tasks || tasks.length === 0) {
        grid.innerHTML = '';
        emptyState.classList.remove('hidden');
        return;
    }

    emptyState.classList.add('hidden');
    grid.innerHTML = tasks.map(task => createTaskCardHTML(task)).join('');
}

// Generate HTML for a single task card
function createTaskCardHTML(task) {
    const dueDateStr = task.due_date 
        ? new Date(task.due_date).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
        : 'No due date';

    const isOverdue = task.due_date && new Date(task.due_date) < new Date() && task.status !== 'completed';

    return `
        <article class="task-card priority-${task.priority}" id="task-card-${task.id}">
            <div class="task-header">
                <h3 class="task-title">${escapeHtml(task.title)}</h3>
                <div class="task-badges">
                    <span class="badge badge-status-${task.status}">${formatStatusLabel(task.status)}</span>
                </div>
            </div>

            ${task.description ? `<p class="task-description">${escapeHtml(task.description)}</p>` : ''}

            <div class="task-badges">
                <span class="badge badge-category">📁 ${escapeHtml(task.category || 'General')}</span>
                <span class="badge badge-priority">🔥 ${escapeHtml(task.priority.toUpperCase())}</span>
            </div>

            <div class="task-footer">
                <div class="task-due-date ${isOverdue ? 'overdue' : ''}">
                    📅 ${dueDateStr} ${isOverdue ? '<span style="color:var(--danger);font-weight:600;">(Overdue)</span>' : ''}
                </div>
                <div class="task-actions">
                    <button class="btn-icon" title="Toggle Status" onclick="quickToggleStatus(${task.id}, '${task.status}')">
                        ${task.status === 'completed' ? '↩️' : '✅'}
                    </button>
                    <button class="btn-icon" title="Edit Task" onclick="openTaskModal(${task.id})">✏️</button>
                    <button class="btn-icon" title="Delete Task" onclick="deleteTask(${task.id})">🗑️</button>
                </div>
            </div>
        </article>
    `;
}

// Format status labels
function formatStatusLabel(status) {
    switch (status) {
        case 'todo': return 'To Do';
        case 'in_progress': return 'In Progress';
        case 'completed': return 'Completed';
        default: return status;
    }
}

// Update Dashboard Statistics Cards
function updateStats(tasks) {
    const total = tasks.length;
    const todo = tasks.filter(t => t.status === 'todo').length;
    const progress = tasks.filter(t => t.status === 'in_progress').length;
    const completed = tasks.filter(t => t.status === 'completed').length;

    document.getElementById('stat-total').textContent = total;
    document.getElementById('stat-todo').textContent = todo;
    document.getElementById('stat-progress').textContent = progress;
    document.getElementById('stat-completed').textContent = completed;
}

// Debounce search input
function debounceSearch() {
    clearTimeout(searchDebounceTimer);
    searchDebounceTimer = setTimeout(() => {
        loadTasks();
    }, 300);
}

// Task Modal Controls
function openTaskModal(taskId = null) {
    const modal = document.getElementById('task-modal');
    const modalTitle = document.getElementById('modal-title');
    const taskIdInput = document.getElementById('task-id');
    const titleInput = document.getElementById('task-title-input');
    const descInput = document.getElementById('task-description-input');
    const statusInput = document.getElementById('task-status-input');
    const priorityInput = document.getElementById('task-priority-input');
    const categoryInput = document.getElementById('task-category-input');
    const dueDateInput = document.getElementById('task-duedate-input');

    if (taskId) {
        const task = allTasks.find(t => t.id === taskId);
        if (!task) return;

        modalTitle.textContent = 'Edit Task';
        taskIdInput.value = task.id;
        titleInput.value = task.title;
        descInput.value = task.description || '';
        statusInput.value = task.status;
        priorityInput.value = task.priority;
        categoryInput.value = task.category || 'General';
        
        if (task.due_date) {
            const d = new Date(task.due_date);
            dueDateInput.value = d.toISOString().split('T')[0];
        } else {
            dueDateInput.value = '';
        }
    } else {
        modalTitle.textContent = 'Create New Task';
        taskIdInput.value = '';
        titleInput.value = '';
        descInput.value = '';
        statusInput.value = 'todo';
        priorityInput.value = 'medium';
        categoryInput.value = 'General';
        dueDateInput.value = '';
    }

    modal.classList.remove('hidden');
    titleInput.focus();
}

function closeTaskModal() {
    document.getElementById('task-modal').classList.add('hidden');
}

function closeModalOnBackdrop(event) {
    if (event.target.id === 'task-modal') {
        closeTaskModal();
    }
}

// Handle Form Submission (Create or Update)
async function handleSaveTask(event) {
    event.preventDefault();
    const taskId = document.getElementById('task-id').value;
    const title = document.getElementById('task-title-input').value.trim();
    const description = document.getElementById('task-description-input').value.trim();
    const status = document.getElementById('task-status-input').value;
    const priority = document.getElementById('task-priority-input').value;
    const category = document.getElementById('task-category-input').value.trim() || 'General';
    const dueDateVal = document.getElementById('task-duedate-input').value;

    const payload = {
        title,
        description,
        status,
        priority,
        category,
        due_date: dueDateVal ? dueDateVal : null
    };

    const saveBtn = document.getElementById('save-task-btn');

    try {
        saveBtn.disabled = true;

        if (taskId) {
            await apiRequest(`/tasks/${taskId}`, {
                method: 'PUT',
                body: payload,
            });
            showToast('Task updated successfully', 'success');
        } else {
            await apiRequest('/tasks', {
                method: 'POST',
                body: payload,
            });
            showToast('Task created successfully', 'success');
        }

        closeTaskModal();
        loadTasks();
    } catch (err) {
        showToast(err.message || 'Failed to save task', 'error');
    } finally {
        saveBtn.disabled = false;
    }
}

// Quick status toggle (cycle between todo -> in_progress -> completed -> todo)
async function quickToggleStatus(taskId, currentStatus) {
    let nextStatus = 'in_progress';
    if (currentStatus === 'todo') nextStatus = 'in_progress';
    else if (currentStatus === 'in_progress') nextStatus = 'completed';
    else nextStatus = 'todo';

    try {
        await apiRequest(`/tasks/${taskId}/status`, {
            method: 'PATCH',
            body: { status: nextStatus },
        });
        showToast(`Status updated to ${formatStatusLabel(nextStatus)}`, 'info');
        loadTasks();
    } catch (err) {
        showToast(err.message || 'Failed to update status', 'error');
    }
}

// Delete Task with confirmation
async function deleteTask(taskId) {
    if (!confirm('Are you sure you want to delete this task?')) return;

    try {
        await apiRequest(`/tasks/${taskId}`, {
            method: 'DELETE',
        });
        showToast('Task deleted', 'info');
        loadTasks();
    } catch (err) {
        showToast(err.message || 'Failed to delete task', 'error');
    }
}
