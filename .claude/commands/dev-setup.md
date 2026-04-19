---
name: "dev-setup"
description: Bootstrap full local dev env — Docker infra (Postgres/Redis/MinIO), backend (migrations auto on startup), frontend. Verifies health on localhost:3000/3001/9011.
allowed-tools: Bash(docker:*), Bash(docker compose:*), Bash(cd backend:*), Bash(cd frontend:*), Bash(go:*), Bash(make:*), Bash(pnpm:*), Bash(curl:*), Bash(test:*), Bash(which:*), Bash(cp:*), Read, Edit
---

Set up the full local development environment from scratch. This ensures everything is running and healthy.

**What this command does:**

1. **Infrastructure** — starts Docker containers (Postgres, Redis, MinIO)
2. **Backend** — installs tools, sets up env, runs migrations, starts server
3. **Frontend** — installs deps, starts dev server
4. **Health check** — verifies all services are responding

**Steps**

1. **Start infrastructure**
   ```bash
   docker compose up -d
   ```
   Wait for containers to be healthy. Verify:
   ```bash
   docker compose ps
   ```
   All 3 containers (postgres, redis, minio) must be running.

2. **Check backend .env**
   ```bash
   test -f backend/.env || (cd backend && cp .env.example .env)
   ```
   Verify required variables are set:
   - `DATABASE_URL` — postgres connection string
   - `REDIS_URL` — redis host:port
   - `JWT_SECRET` — >= 32 chars
   - `BACKEND_KEK` — base64 encoded, >= 32 bytes
   - `MINIO_*` — endpoint, access key, secret key, bucket

   If any required variable is empty or has placeholder value, prompt the user to fill them in.

3. **Install backend tools** (if not already installed)
   ```bash
   cd backend && which golangci-lint >/dev/null 2>&1 || make install-tools
   ```

4. **Backend tidy and build**
   ```bash
   cd backend && go mod tidy
   ```
   Report if any dependency issues.

5. **Start backend**
   ```bash
   cd backend && make dev
   ```
   Wait for startup, then verify:
   ```bash
   curl -s http://localhost:3001/health | jq .
   ```
   Health check should return `{"status": "ok"}` with DB and Redis connected.

6. **Install frontend dependencies**
   ```bash
   cd frontend && pnpm install
   ```

7. **Start frontend**
   ```bash
   cd frontend && pnpm dev
   ```
   Verify it starts on port 3000.

8. **Final health summary**
   ```
   Service          URL                  Status
   Postgres         localhost:5432       running
   Redis            localhost:6379       running
   MinIO            localhost:9000       running
   Backend API      http://localhost:3001  ok
   Frontend         http://localhost:3000  ok
   Swagger          http://localhost:3001/docs  ok
   MinIO Console    http://localhost:9001  ok
   ```

**Output**

```
## Dev Setup Complete

All services are running. Access:
- Frontend: http://localhost:3000
- Backend API: http://localhost:3001
- Swagger Docs: http://localhost:3001/docs
- MinIO Console: http://localhost:9001
```

If any service fails to start, report the error with troubleshooting steps.
