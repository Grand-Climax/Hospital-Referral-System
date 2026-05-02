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
- **🧠 ML-Powered Triage**: Integration-ready structures for machine learning predictions on patient severity and scheduling urgency.
- **📅 Dynamic Capacity**: Automated scheduling service with configurable buffers, aging factors, and overbook management.

---

## 🛠️ Technical Stack

- **Language**: Go (Golang) for high-concurrency performance.
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
- **Redis** (Active instance)

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

# Storage (Cloudinary)
CLOUDINARY_CLOUD_NAME=your_cloud_name
CLOUDINARY_API_KEY=your_api_key
CLOUDINARY_API_SECRET=your_api_secret

# SMS (AfroMessage)
AFROMESSAGE_API_KEY=your_afromessage_key
AFROMESSAGE_SENDER_NAME=your_sender_name
AFROMESSAGE_IDENTIFIER_ID=your_id
AFROMESSAGE_BASE_URL=https://api.afromessage.com
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

### Integration & E2E Test Suite
The project maintains a professional test suite covering complex State Machine transitions and Auth flows:

```bash
# Run unit & integration tests
go test ./test -v -count=1

# Run comprehensive E2E suite (Server must be running)
go run scratch/run_e2e_tests_v2/run_e2e_tests_v2.go
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

All accounts use the default password: **`password123`**

### 🌍 System Global
| Role | Email | Name |
| :--- | :--- | :--- |
| **Super Admin** | `superadmin@moh.gov.et` | System Super Admin |
| **MoH Analyst** | `analyst@moh.gov.et` | MoH Analyst |

### 🏥 Tikur Anbessa Specialized Hospital
| Role | Email | Name |
| :--- | :--- | :--- |
| **Referring Doctor** | `doctor.ta@hospital.et` | Alemayehu Doctor |
| **Referring Doctor** | `doc.primary@hospital.et` | Primary Doc |
| **Liaison Officer** | `liaison.ta@hospital.et` | Sara Liaison |
| **Liaison Officer** | `liaison@moh.gov.et` | MoH Liaison |
| **Specialist** | `specialist.ta@hospital.et` | Yohannes Specialist |
| **Specialist** | `specialist.cardio@hospital.et` | Cardio Specialist |
| **Receptionist** | `reception.ta@hospital.et` | Aster Receptionist |
| **Receptionist** | `reception.primary@hospital.et` | Primary Reception |
| **Dept Head** | `depthead.ta@hospital.et` | Genet Dept Head |

### 🏥 St. Paul's Hospital (SPHMMC)
| Role | Email | Name |
| :--- | :--- | :--- |
| **Hospital Admin** | `admin.specialized@hospital.et` | Hospital Admin |
| **Referring Doctor** | `doctor.sp@hospital.et` | Tesfaye Doctor |
| **Liaison Officer** | `liaison.sp@hospital.et` | Liaison SP |
| **Specialist** | `specialist.sp@hospital.et` | Kidist Specialist |
| **Receptionist** | `reception.sp@hospital.et` | Etagegn Receptionist |
| **Dept Head** | `depthead.sp@hospital.et` | Henok Dept Head |

### 🏥 Black Lion Hospital
| Role | Email | Name |
| :--- | :--- | :--- |
| **Referring Doctor** | `doctor.bl@hospital.et` | Doctor BL |
| **Liaison Officer** | `liaison.bl@hospital.et` | Liaison BL |
| **Specialist** | `specialist.bl@hospital.et` | Martha Specialist |
| **Receptionist** | `reception.bl@hospital.et` | Frehiwot Receptionist |
| **Dept Head** | `depthead.bl@hospital.et` | Head BL |

---

## 📂 Repository Structure

```text
├── cmd/                # Entry points (server, seeder)
├── internal/           # Private application code
│   ├── delivery/       # HTTP handlers, DTOs, and routes
│   ├── domain/         # Entities and interfaces
│   ├── infrastructure/ # DB, Redis, Caching, Middleware, Storage
│   ├── repository/     # Data persistence logic
│   ├── seeds/          # Database seeders
│   └── usecase/        # Business logic orchestration
├── pkg/                # Reusable public packages
├── docs/               # Auto-generated Swagger docs
├── test/               # Integration tests
└── scratch/            # Utility scripts and E2E runs
```
