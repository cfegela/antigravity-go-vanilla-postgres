# ⚡ TaskFlow - Containerized Go + Postgres + Vanilla JS CRUD Application

A modern, high-performance, containerized full-stack CRUD web application for Task & Project Management. Built with a Go REST API, PostgreSQL database, Vanilla JS/CSS frontend, and orchestrated via Docker Compose or Podman.

Antigravity wrote this in less than 5 minutes using Gemini Flash 3.6. God help us all.

---

## 🌟 Key Features

- **🔒 Modern Authentication**: Secure JWT-based auth with Access & Refresh tokens stored in **HttpOnly Cookies** (protected against XSS attacks) and server-side token revocation.
- **🚀 Go 1.22 REST API**: Standard library `net/http` pattern routing, bcrypt password hashing, CORS middleware, and PostgreSQL database integration.
- **🗄️ PostgreSQL Database**: Auto-initialized schema with tables for `users`, `refresh_tokens`, and `tasks`. Indexed for high-performance querying.
- **🎨 Modern Vector Icon System**: Integrated high-resolution SVG vector icon helper (`js/icons.js`) replacing retro 1990s clip-art emojis across brand logos, dashboard statistics, category & status badges, priority indicators, due dates, action buttons, tech badges, and empty states.
- **💎 Glassmorphic & Modern CSS UI**: Clean responsive layout built with HTML5 and CSS3 (Inter/Outfit Google typography, dark blue navbar, custom badges, and smooth action micro-animations).
- **⚡ Container Orchestration**: Single command container deployment (`docker compose up --build` or `podman compose up --build`) featuring health checks and Nginx reverse proxying (`/api/*`).

---

## 📂 Project Architecture

```
antigravity-go-vanilla-postgres/
├── docker-compose.yml        # Docker / Podman Compose configuration (db, backend, frontend)
├── .env                      # Environment configuration
├── README.md                 # Project documentation
├── backend/                  # Go REST API Backend
│   ├── Dockerfile            # Multi-stage Docker build for Go
│   ├── go.mod                # Go module specification
│   ├── main.go               # Server entry point & CORS
│   ├── db/
│   │   ├── db.go             # PostgreSQL connection pool & auto migrations
│   │   └── init.sql          # DB Schema definitions & indexes
│   ├── auth/
│   │   └── auth.go           # JWT token generation, verification & bcrypt
│   ├── middleware/
│   │   └── auth_middleware.go# Request authentication context middleware
│   └── handlers/
│       ├── auth_handler.go   # Register, Login, Refresh, Logout, /me
│       └── task_handler.go   # Tasks CRUD handlers (List, Create, Update, Delete, Patch)
└── frontend/                 # Vanilla JS/CSS Frontend
    ├── Dockerfile            # Nginx container setup
    ├── nginx.conf            # Nginx config & API reverse proxy
    ├── index.html            # Main SPA HTML structure with SVG icon elements
    ├── css/
    │   └── styles.css        # Modern design system & SVG icon system styles
    └── js/
        ├── api.js            # HTTP client & silent token refresh handler
        ├── auth.js           # Auth UI & session management
        ├── icons.js          # Modern SVG vector icon generator system
        └── app.js            # Task CRUD DOM interactions & statistics
```

---

## 🚀 Quick Start (Running Locally via Docker or Podman)

### Prerequisites
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) or [Podman](https://podman.io/) installed and running on your system.

### Running the Application

1. **Clone or navigate to the directory**:
   ```bash
   cd antigravity-go-vanilla-postgres
   ```

2. **Start the containers**:
   ```bash
   # Using Docker:
   docker compose up --build

   # Or using Podman:
   podman compose up --build
   ```

3. **Access the application**:
   - 🌐 **Web Frontend**: Open [http://localhost:8080](http://localhost:8080) in your browser.
   - 📡 **Go Backend API**: Direct API accessible at [http://localhost:8081/api/health](http://localhost:8081/api/health) or proxied via [http://localhost:8080/api/health](http://localhost:8080/api/health).
   - 🐘 **PostgreSQL**: Connected internally on port `5432` (or exposed on `localhost:5432`).

---

## 📡 API Endpoints Specification

### Authentication Endpoints
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :---: |
| `POST` | `/api/auth/register` | Register new user & set HttpOnly cookies | ❌ |
| `POST` | `/api/auth/login` | Authenticate user & set HttpOnly cookies | ❌ |
| `POST` | `/api/auth/refresh` | Silent refresh of expired Access Token | ❌ (Cookie) |
| `POST` | `/api/auth/logout` | Revoke refresh token & clear cookies | ❌ |
| `GET` | `/api/auth/me` | Fetch authenticated user profile | ✅ |

### Task CRUD Endpoints
| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :---: |
| `GET` | `/api/tasks` | List user tasks (Supports query params: `status`, `priority`, `q`) | ✅ |
| `POST` | `/api/tasks` | Create a new task | ✅ |
| `GET` | `/api/tasks/{id}` | Get single task details | ✅ |
| `PUT` | `/api/tasks/{id}` | Update existing task | ✅ |
| `PATCH`| `/api/tasks/{id}/status`| Quick status toggle (`todo`, `in_progress`, `completed`)| ✅ |
| `DELETE`|`/api/tasks/{id}` | Delete a task | ✅ |

---

## 🛠️ Testing Local Build without Containers (Optional)

To test the backend locally without containers (requires a local PostgreSQL instance running):
```bash
cd backend
export DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=taskdb JWT_SECRET=testsecret
go run main.go
```
Then serve the `frontend/` directory with any static HTTP server (e.g. `npx serve frontend`).
