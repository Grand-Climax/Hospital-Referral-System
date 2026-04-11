# 🏥 Hospital Referral Hub - Backend

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Framework](https://img.shields.io/badge/Framework-Gin-008ECF?style=flat)](https://gin-gonic.com/)
[![ORM](https://img.shields.io/badge/ORM-GORM-00ADD8?style=flat)](https://gorm.io/)
[![Database](https://img.shields.io/badge/Database-PostgreSQL-336791?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Cache](https://img.shields.io/badge/Cache-Redis-DC382D?style=flat&logo=redis)](https://redis.io/)

A mission-critical digital infrastructure for standardized patient referrals within the Ethiopian healthcare ecosystem. This backend service orchestrates complex referral workflows, enforces granular role-based security, and maintains an immutable audit trail of the patient journey.

---

## 🏗️ Core Architecture

The system is built on **Clean Architecture** principles, ensuring a strict separation of concerns and high testability.

- **Domain Layer**: Pure business entities and interface definitions.
- **UseCase Layer**: Orchestration of business logic and state transitions.
- **Repository Layer**: Data persistence (PostgreSQL) and caching (Redis) abstractions.
- **Delivery Layer**: RESTful API implementation using the Gin framework.

### State-Driven Workflows
At its heart, the system manages a robust **Referral State Machine**:
`PENDING` → `REVIEWING` → `ACCEPTED/REJECTED` → `COMPLETED`

---

## ✨ Key Features

- **🔐 Granular RBAC**: 8+ distinct healthcare roles (Doctors, Specialists, Admins, Liaisons, etc.) with a strictly enforced permission matrix.
- **⚡ Dual-Layer Auth**: JWT-based authentication with high-performance session mirroring in Redis for sub-millisecond validation.
- **📂 State Machine & History**: Automated tracking of every status change with an attached audit log for medical accountability.
- **📡 Multi-Hospital Networking**: Intelligent routing of referrals between Tertiary, General, and Primary healthcare tiers.
- **💾 Attachment Handling**: Support for medical documents, images, and DICOM files linked to clinical cases.
- **📊 Real-time Audit**: Immutable logs capturing WHO, WHAT, and WHEN for every critical system interaction.

---

## 🛠️ Technical Stack

- **Lanuage**: Go (Golang) for high-concurrency performance.
- **Framework**: [Gin Gonic](https://gin-gonic.com/) for high-performance HTTP routing.
- **Persistence**: [PostgreSQL](https://www.postgresql.org/) managed via [GORM](https://gorm.io/).
- **Caching**: [Redis](https://redis.io/) for session management, logout blacklisting, and rate limiting.
- **Security**: JWT (Access/Refresh), Bcrypt for password hashing.
- **Documentation**: [Swagger (Swag)](https://github.com/swaggo/swag) for automated OpenAPI-compliant documentation.

---

## 🚀 Getting Started

### 1. Prerequisites
- **Go** (1.20+)
- **PostgreSQL** (Active instance)
- **Docker** (Recommended for Redis)

### 2. Environment Configuration
Configure your local environment by creating a `.env` file at the project root:

```bash
# Database
DATABASE_URL=postgres://user:password@localhost:5432/referral_db

# Caching & Session
REDIS_URL=localhost:6379

# Security
JWT_SECRET=your_super_secret_key_here
PORT=8081
```

### 3. Infrastructure (Redis)
Start the Redis container via Docker:

```bash
docker run -d --name referral-redis -p 6379:6379 redis
```

---

## 🏃 Operation Commands

### 🧊 Database Seeding
Populate the system with Ethiopian regional hospitals, departments, and pre-configured test roles:

```bash
go run cmd/seeder/main.go
```

### 🔥 Start Server
Launch the API server in debug mode:

```bash
go run cmd/server/main.go
```

---

## 🧪 Development & Quality

### Integration Test Suite
The project maintains a professional test suite covering complex State Machine transitions and Auth flows:

```bash
go test ./test -v -count=1
```

### API Exploration (Swagger)
The live documentation is available at:
**`http://localhost:8081/swagger/index.html`**

To regenerate documentation:
```bash
swag init -g cmd/server/main.go --output docs
```

---

## 🔑 Pre-configured Test Accounts

| Role | Email | Use Case |
| :--- | :--- | :--- |
| **Referring Doctor** | `doc.primary@hospital.et` | Initiate referrals |
| **Specialist** | `specialist.cardio@hospital.et` | Review/Accept cases |
| **Receptionist** | `reception.primary@hospital.et` | Confirm patient arrival |
| **Liaison Officer** | `liaison@moh.gov.et` | Oversee regional routing |
| **Hospital Admin** | `admin.specialized@hospital.et` | Manage hospital resources |
| **MoH Analyst** | `analyst@moh.gov.et` | View global statistics |
| **System Admin** | `superadmin@moh.gov.et` | Global configuration |

*Default password for all accounts: **`password123`***

---

## 📂 Repository Structure

```text
├── cmd/                # Entry points (server, seeder)
├── internal/           # Private application code
│   ├── delivery/       # HTTP handlers and routes
│   ├── domain/         # Entities and interfaces
│   ├── infrastructure/ # DB, Redis, Middleware
│   ├── repository/     # Data persistence logic
│   └── usecase/        # Business logic orchestration
├── pkg/                # Reusable public packages
├── docs/               # Auto-generated Swagger docs
└── test/               # Integration & E2E tests
```
