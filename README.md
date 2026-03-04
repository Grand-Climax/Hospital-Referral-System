# Hospital Referral Hub - Backend

A digital platform for patient referrals in Ethiopian healthcare. This repository contains the backend service built with Go (Golang), utilizing PostgreSQL for persistent storage and Redis for session management and rate-limiting.

---

## 🚀 Current Implementation Status (Sprints 1-3)

The backend is structured according to **Clean Architecture** principles and currently fully satisfies the goals up to Sprint 3:

*   **Sprint 1**: 
    *   Foundational project structure (`cmd`, `internal`, `pkg`, `config`, `docs`).
    *   Docker configuration with multi-stage builds (`Dockerfile`, `.dockerignore`).
    *   API routing setup (Gin framework) and basic middlewares (CORS, Logging, Recovery).
*   **Sprint 2**: 
    *   Core Database Schema designed and mapped via GORM (`users`, `hospitals`, `departments`, `referrals`, `referral_status_history`, `audit_logs`).
    *   Role-Based capability model securely defined in `internal/pkg/auth/permissions.go` (The Permission Matrix).
    *   Automated seeding script (`cmd/seeder`) to populate Ethiopian regional hospitals, departments, and 7 test user roles.
*   **Sprint 3**: 
    *   Secure Authentication: Login endpoint with `bcrypt` password verification, issuing JWT Access (15m) and Refresh Tokens (7d).
    *   Session Management: Secure session mirroring to both PostgreSQL and Redis cache for lightning-fast validation.
    *   RBAC Middleware: Requests are intercepted, JWTs parsed, and checked against the predefined Permission Matrix (`RequirePermission` middleware).
    *   Rate Limiting (max 5 login requests/min/IP) and secure Logout (Access token Blacklisting in Redis).

---

## 🛠️ Prerequisites

To run this project locally, you must have the following installed:
1.  **Go** (1.20+ recommended)
2.  **PostgreSQL** (Active database named `referral` or `hospital_referral`)
3.  **Docker Desktop** (Required for easily running Redis)

---

## Environment Setup

The application requires specific environment variables to connect to PostgreSQL, Redis, and secure the JWTs. 

1. At the root of your project, create a file named `.env.local` (or copy the provided example):
```bash
cp .env.example .env.local
```

2. Open `.env.local` and configure your keys. *Note: `.env.local` is ignored by git for security.*

**Available Keys Explained:**
*   `DATABASE_URL`: Your PostgreSQL connection string. Must include your local postgres password and the exact database name you created.
*   `REDIS_URL`: The host and mapped port of your Redis instance (usually `localhost:6379`).
*   `JWT_SECRET`: A secure, secret string used to cryptographically sign your authentication tokens.
*   `PORT`: The port the Go server will listen on (default `8081`).

---

## 🐳 Running Redis via Docker

The application completely relies on Redis for fast session lookups and rate-limiting. The easiest way to run Redis locally without installing it on your OS is through Docker.

1. **Ensure Docker is Running**: Open Docker Desktop.
2. **Start a Redis Container**: Open your terminal and run the following command to download the Redis image and run a container named `my-redis` mapped securely to port `6379`:
    ```bash
    docker run -d --name my-redis -p 6379:6379 redis
    ```
    *   `-d`: Runs the container in the background (detached).
    *   `--name my-redis`: Names the container so you can easily reference it later.
    *   `-p 6379:6379`: **Crucial Mapping**. This maps your computer's local port 6379 directly to the container's internal port 6379, allowing Go to connect to `localhost:6379`.
3. **Verify it is running**: 
    ```bash
    docker ps
    ```
    You should see `my-redis` in the list with `0.0.0.0:6379->6379/tcp`.

---

## 🏃‍♂️ Running the Application

Once your `.env.local` is configured, PostgreSQL is active, and the Docker Redis container is running:

### 1. Seed the Database
Before starting the server, you need to populate your PostgreSQL database with the required tables, hospitals, and test users.
```bash
go run cmd/seeder/main.go
```
*If successful, the terminal will print `Finished executing database seeders!`*

### 2. Start the Server
Start the Gin HTTP server:
```bash
go run cmd/server/main.go
```
*If successful, the terminal will show that it connected to the database, Redis, and is listening on `:8081`.*

---

## 🔑 Test Users for Authentication

The seeder creates users for all 7 roles with the default password: **`password123`**

You can test the `/api/v1/auth/login` endpoint using an API client (like Postman) with these emails:
*   `doc.primary@hospital.et` (Referring Doctor)
*   `specialist.cardio@hospital.et` (Receiving Specialist)
*   `head.cardio@hospital.et` (Department Head)
*   `admin.specialized@hospital.et` (Hospital Admin)
*   `superadmin@moh.gov.et` (System Super Admin)
*   `analyst@moh.gov.et` (MoH Analyst)
*   `liaison@moh.gov.et` (Regional Liaison Officer)
*   `reception.primary@hospital.et` (Receptionist)

---

## 📜 Clean Architecture Guide
To maintain the integrity of this codebase, follow these rules when adding new features:
1.  **Define Entities First** (`internal/domain/entity`): Put your core structs, enums, and database schema representations here.
2.  **Define Repositories** (`internal/repository`): Create interfaces and Postgres implementations for database CRUD operations.
3.  **Define UseCases** (`internal/usecase`): Business logic lives here. UseCases should only communicate with Repositories and external services, never directly with HTTP or Gin.
4.  **Define Handlers** (`internal/delivery/http/handlers`): Attach HTTP routes to UseCases and handle JSON binding/responses.
