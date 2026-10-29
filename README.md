# Chit Fund Management API v2

Production-ready REST API for multi-tenant chit fund management built with Go.

## What's New in v2

Full implementation of all 75+ endpoints across 11 modules — replacing stubs with complete repository, service, and handler layers.

## Tech Stack

- **Go 1.21+** | **PostgreSQL 15** | **Redis 7** | **Gin** | **JWT** | **Zerolog** | **Docker**

## Quick Start

```bash
cp .env.example .env          # configure your DB/Redis/JWT
make docker-up                # spin up postgres + redis
make migrate-up               # run schema migrations
make run                      # start the API on :8080
```

Swagger UI: http://localhost:8080/swagger/index.html

## Project Structure

```
chit-fund-api/
├── cmd/api/main.go                  # Entry point — all routes wired here
├── internal/
│   ├── config/                      # Viper config loader
│   ├── domain/                      # Entities + request/response types
│   │   ├── requests.go              # All request/response structs
│   │   ├── company.go / user.go / subscriber.go
│   │   ├── chit_group.go / auction.go / installment.go / payout.go
│   │   ├── branch.go                # Branch, State, Role
│   │   ├── kyc.go                   # SubscriberDocument
│   │   ├── defaults.go              # Default, LegalNotice, AuditLog
│   │   └── dashboard.go             # KPIs, reports, ledger
│   ├── handler/                     # HTTP handlers (13 files)
│   ├── middleware/                  # Auth, RBAC, rate-limit, tenant, CORS
│   ├── repository/                  # PostgreSQL data access (15 files)
│   └── service/                     # Business logic (13 service types)
├── pkg/
│   ├── cache/redis.go
│   ├── database/postgres.go
│   ├── jwt/jwt.go
│   └── logger/logger.go
├── migrations/
├── docs/                            # Swagger (regenerate with: make swagger)
├── docker-compose.yaml
├── Dockerfile
└── Makefile
```

## API Endpoints (75 total)

### Auth  `/api/v1/auth`
| Method | Path | Description |
|--------|------|-------------|
| POST | /login | Staff login — returns JWT pair |
| POST | /logout | Invalidate session (Bearer required) |
| POST | /refresh | Refresh access token (no auth) |
| GET  | /me | Get current user profile |
| POST | /change-password | Change own password |

### Dashboard  `/api/v1/dashboard`
| Method | Path | Description |
|--------|------|-------------|
| GET | /dashboard | Full dashboard bundle (KPIs + chart) |
| GET | /dashboard/kpis | KPI cards only — lightweight |
| GET | /dashboard/collections | Monthly collection chart data |

### Company & Configuration
| Method | Path | Description |
|--------|------|-------------|
| GET | /company | Get company profile |
| PUT | /company | Update company details |
| GET | /branches | List all branches |
| POST | /branches | Create new branch |
| GET | /branches/:id | Get branch by ID |
| PUT | /branches/:id | Update branch |
| GET | /states | List states with regulatory config |
| GET | /states/:id | Get state regulatory values |

### Users & Access
| Method | Path | Description |
|--------|------|-------------|
| GET | /users | List staff users (paginated) |
| POST | /users | Create staff user |
| GET | /users/:id | Get user by ID |
| PUT | /users/:id | Update user |
| PATCH | /users/:id/deactivate | Deactivate user account |
| PATCH | /users/:id/activate | Re-activate user account |
| GET | /roles | List all roles |
| GET | /roles/:id | Get role with permissions |
| PUT | /roles/:id | Update role permissions |

### Subscribers & KYC
| Method | Path | Description |
|--------|------|-------------|
| GET | /subscribers | List subscribers (paginated + filter) |
| POST | /subscribers | Create subscriber |
| GET | /subscribers/:id | Get subscriber profile |
| PUT | /subscribers/:id | Update subscriber |
| PATCH | /subscribers/:id/blacklist | Blacklist subscriber |
| PATCH | /subscribers/:id/unblacklist | Remove blacklist |
| GET | /subscribers/:id/documents | List KYC documents |
| POST | /subscribers/:id/documents | Upload KYC document |
| PATCH | /subscribers/:id/documents/:docId/verify | Verify KYC document |
| GET | /subscribers/:id/banks | List bank accounts |
| POST | /subscribers/:id/banks | Add bank account |
| DELETE | /subscribers/:id/banks/:bankId | Remove bank account |

### Chit Groups
| Method | Path | Description |
|--------|------|-------------|
| GET | /chit-groups | List chit groups (paginated + filter) |
| POST | /chit-groups | Create chit group |
| GET | /chit-groups/:id | Get chit group details |
| PUT | /chit-groups/:id | Update chit group |
| PATCH | /chit-groups/:id/close | Close/terminate chit group |
| GET | /chit-groups/:id/subscribers | List enrolled subscribers |
| POST | /chit-groups/:id/subscribers | Enroll subscriber into group |
| DELETE | /chit-groups/:id/subscribers/:csId | Remove enrollment |

### Auctions
| Method | Path | Description |
|--------|------|-------------|
| GET | /auctions | List auctions (filter by group/status) |
| POST | /auctions | Schedule auction |
| GET | /auctions/:id | Get auction details |
| PUT | /auctions/:id | Update auction details |
| PATCH | /auctions/:id/conduct | Mark auction as conducted |
| PATCH | /auctions/:id/post-result | Post winner + discount |
| GET | /auctions/:id/bids | List bids for auction |
| POST | /auctions/:id/bids | Submit bid |
| PATCH | /auctions/:id/bids/:bidId/win | Mark bid as winner |

### Installments & Payments
| Method | Path | Description |
|--------|------|-------------|
| GET | /installments | List installments (filter by group/month) |
| POST | /installments/generate | Generate monthly installment batch |
| GET | /installments/:id | Get installment details |
| GET | /subscriber-installments | List subscriber-level installments |
| GET | /subscriber-installments/:id | Get subscriber installment |
| GET | /payment-transactions | List payment transactions |
| POST | /payment-transactions | Record payment |
| GET | /payment-transactions/:id | Get transaction details |

### Payouts & Dividends
| Method | Path | Description |
|--------|------|-------------|
| GET | /payouts | List payouts (filter by status/group) |
| POST | /payouts | Initiate payout for winner |
| GET | /payouts/:id | Get payout details |
| PATCH | /payouts/:id/approve | Approve payout (manager) |
| PATCH | /payouts/:id/disburse | Mark as disbursed |
| GET | /dividends | List dividend records |
| POST | /dividends/calculate | Calculate dividend for auction |
| POST | /dividends/post | Post dividend to installments |

### Defaults & Legal
| Method | Path | Description |
|--------|------|-------------|
| GET | /defaults | List defaults (filter by status/branch) |
| POST | /defaults/detect | Run default detection job |
| PATCH | /defaults/:id/flag | Manually flag as default |
| PATCH | /defaults/:id/resolve | Mark default as resolved |
| POST | /defaults/:id/reminder | Send payment reminder |
| GET | /legal-notices | List legal notices |
| POST | /legal-notices | Issue legal notice |
| GET | /legal-notices/:id | Get legal notice |
| PATCH | /legal-notices/:id/escalate | Escalate to court summon |

### Reports & Audit
| Method | Path | Description |
|--------|------|-------------|
| GET | /reports/monthly-collections | Monthly collection summary |
| GET | /reports/defaulters | Defaulters report |
| GET | /reports/chit-performance | Chit group performance |
| GET | /reports/subscriber-ledger/:id | Subscriber ledger |
| GET | /audit-logs | Audit log (paginated + filter) |
| GET | /system-logs | System error logs |

## Authentication

All protected endpoints require:
```
Authorization: Bearer <access_token>
```

Tokens are JWT signed with HS256. Access token expires in 15 minutes, refresh token in 24 hours.

## Multi-Tenancy

Every repository query filters by `company_id` extracted from the JWT. No data leaks between tenants.

## Development

```bash
make dev          # hot reload with Air
make test         # run tests
make swagger      # regenerate docs (requires swag CLI)
make migrate-create name=add_table   # new migration
```

## Configuration

Priority: Environment variables > `.env` file > `config.yaml`

Key variables:
```
DATABASE_HOST, DATABASE_USER, DATABASE_PASSWORD, DATABASE_DBNAME
REDIS_HOST, REDIS_PORT
JWT_SECRET, JWT_ACCESSTOKENTTL, JWT_REFRESHTOKENTTL
ENVIRONMENT (development|production)
```
