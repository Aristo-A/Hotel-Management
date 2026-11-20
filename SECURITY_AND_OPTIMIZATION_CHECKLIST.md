# Hotel Management System – Security, Performance & Optimization Checklist

> **Authoritative Engineering Standards & Production Readiness Specification**  
> **Target Architecture:** Modular Monolith in Go (Gin) + PostgreSQL 16 + GORM + Redis + React 18 / Tailwind + Docker  
> **Localization:** India | Currency: INR (₹) | Default Tax: Configurable GST (12% baseline with CGST/SGST/IGST support)  

---

## Table of Contents

1. [Core Architectural Principle](#core-architectural-principle)
2. [1. Security Checklist (Critical Priority)](#1-security-checklist-critical-priority)
   - [A. Authentication and Authorization](#a-authentication-and-authorization)
   - [B. Multi-Tenant Security (Highest Priority)](#b-multi-tenant-security-highest-priority)
   - [C. API Security](#c-api-security)
   - [D. Database Security](#d-database-security)
   - [E. Payment Security](#e-payment-security)
3. [2. Performance and Optimization Checklist (High Priority)](#2-performance-and-optimization-checklist-high-priority)
   - [A. Backend Optimization & Go Concurrency](#a-backend-optimization--go-concurrency)
   - [B. Database Optimization & Indexing](#b-database-optimization--indexing)
   - [C. Caching Strategy (Redis & CDN)](#c-caching-strategy-redis--cdn)
   - [D. Frontend Optimization](#d-frontend-optimization)
   - [E. Network Optimization](#e-network-optimization)
4. [3. Hotel-Specific Business Logic Checklist](#3-hotel-specific-business-logic-checklist)
   - [A. Booking Management](#a-booking-management)
   - [B. Room Management & State Machine](#b-room-management--state-machine)
   - [C. Payments and Billing](#c-payments-and-billing)
   - [D. GST & Indian Localization Engine](#d-gst--indian-localization-engine)
   - [E. Housekeeping & Hotel Services](#e-housekeeping--hotel-services)
   - [F. Guest Management](#f-guest-management)
5. [4. Multi-Tenant Architecture Specification](#4-multi-tenant-architecture-specification)
6. [5. Testing & Quality Assurance Checklist](#5-testing--quality-assurance-checklist)
   - [A. Functional Testing](#a-functional-testing)
   - [B. Security Testing](#b-security-testing)
   - [C. Performance Testing & Benchmarks](#c-performance-testing--benchmarks)
   - [Suggested Performance Targets](#suggested-performance-targets)
7. [6. DevOps, Deployment & Monitoring Checklist](#6-devops-deployment--monitoring-checklist)
   - [A. Infrastructure & CI/CD](#a-infrastructure--cicd)
   - [B. Production Monitoring & Telemetry](#b-production-monitoring--telemetry)
8. [7. Overlooked Critical Optimizations](#7-overlooked-critical-optimizations)
9. [8. Recommended Development Priorities (Phases 1–6)](#8-recommended-development-priorities-phases-16)
10. [9. Technology and Tooling Stack](#9-technology-and-tooling-stack)
11. [10. Master Feature Release Checklist (Mandatory 31-Point Gate)](#10-master-feature-release-checklist-mandatory-31-point-gate)

---

## Core Architectural Principle

> **Strict Multi-Tenant Isolation:**  
> A hotel must only be able to access and manage its own property's data, while the platform administrator can manage all properties according to their permissions.  
> **Never trust `tenant_id` or `hotel_id` sent by the frontend.** Derive tenant access exclusively from the authenticated user's authorized memberships injected into the request context from cryptographically verified JWT claims.

---

## 1. Security Checklist (Critical Priority)

### A. Authentication and Authorization
- [x] **Secure Password Hashing:** Use `bcrypt` (cost >= 12) or `Argon2id`.
- [x] **No Plain-Text Passwords:** Never log or store plain-text credentials at any layer.
- [x] **Access Tokens:** Issue HMAC-SHA256 (`HS256`) or Ed25519 JWT access tokens with short expiration periods (e.g., 15 minutes to 24 hours max).
- [ ] **Refresh Token Rotation:** Use refresh token rotation and revocation stored in Redis or database.
- [x] **Enforce Backend RBAC:** Never rely on frontend role checks alone. Enforce permissions strictly on the backend.
- [x] **Multi-Role Support:** Implement and verify backend permissions across all defined roles:

| Role | Scope | Description & Permissions |
| :--- | :--- | :--- |
| **Platform Super Admin** | Global Platform | Manage platform, create/suspend hotels, assign hotel admins, view system analytics |
| **Hotel Group Admin** | Hotel Group | Manage multiple properties owned by an assigned hotel brand / group |
| **Hotel Admin** | Single Property | Manage property rooms, categories, live desk operations, billing, and reports |
| **Manager** | Single Property | Manage property staff, room assignments, operational schedules, view reports |
| **Receptionist** | Single Property | Front-desk guest bookings, walk-ins, check-in, check-out, issue key cards |
| **Accountant** | Single Property | Process payments, generate invoices, manage GST filing reports, refunds |
| **Housekeeping** | Single Property | Update room cleaning status (`dirty` -> `cleaning` -> `available`), report issues |
| **Customer / Guest** | Personal Booking | Zero-barrier public booking; view/manage personal reservation via reference code |

- [ ] **MFA:** Support Multi-Factor Authentication (TOTP) for Platform Super Admins and sensitive property-level administrative accounts.
- [x] **Login Rate Limiting:** Implement IP- and email-based rate limiting (e.g., max 5 failed attempts per minute) on `/api/auth/login`.
- [ ] **Account Lockout:** Implement progressive delays or temporary account lockout after repeated failed attempts.
- [ ] **Brute-Force Defense:** Block automated credential stuffing with Redis-backed leaky/token buckets.
- [ ] **Secure Password Reset:** Single-use, cryptographically random, time-limited tokens (max 15 min expiry).
- [x] **Session Invalidation:** Invalidate all active user tokens/sessions immediately upon password change or account deactivation.
- [x] **Authentication Logging:** Log all successful logins, failed attempts, and role privilege elevations in `activity_logs`.
- [ ] **Cookie Security:** If cookie-based auth is used, enforce `HttpOnly`, `Secure`, and `SameSite=Strict` attributes.

---

### B. Multi-Tenant Security (Highest Priority)
- [x] **Tenant Ownership FK:** Every operational table (`rooms`, `room_categories`, `reservations`, `bills`, `bill_items`, `facility_charges`, `payments`, `housekeeping_logs`, `activity_logs`) must contain a non-nullable `hotel_id UUID` foreign key.
- [x] **Context-Driven Tenant Identity:** Validate tenant identity on every API request. Derive `hotel_id` from the authenticated JWT claims:
  ```go
  hotelID := c.GetString("hotelID")
  ```
- [x] **Zero-Trust Frontend:** Never trust `hotel_id` or `tenant_id` provided in request URL params or request JSON bodies.
- [x] **Pre-Query Ownership Verification:** Always scope read, update, and delete queries:
  ```sql
  -- SAFE: Scoped by verified tenant context
  SELECT * FROM reservations WHERE id = $1 AND hotel_id = $2;
  UPDATE rooms SET status = $1 WHERE id = $2 AND hotel_id = $3;
  ```
- [x] **Prevent Cross-Tenant Booking Access:** If a receptionist from Hotel A requests booking `BK-102` belonging to Hotel B, the API **must return 404 Not Found** (preventing enumeration) or `403 Forbidden`.
- [x] **Prevent Cross-Tenant Invoice & Payment Access:** Bills and payments must be joined with `hotel_id`.
- [x] **Prevent Cross-Tenant Customer Data Leakage:** Guests' Aadhaar, PAN, email, and phone records must never be exposed across properties.
- [x] **Prevent Cross-Tenant File & Image Access:** S3 / storage bucket keys must include tenant prefixes (`/hotels/{hotel_id}/rooms/...`) with presigned URL authorization.
- [x] **Tenant Filtering on Background Jobs:** Background tasks (reminders, invoice generation, syncs) must process records tenant by tenant.
- [x] **Tenant Filtering on Reports & Aggregations:** Reports must never aggregate across properties unless requested by a Platform Super Admin.
- [x] **Tenant Context in Cache Keys:** Prefix all Redis cache keys with tenant ID: `hotel:{hotel_id}:rooms`.
- [ ] **Tenant Context in Search Indexes:** Elasticsearch or Postgres full-text search must filter by `hotel_id`.
- [x] **Secure Data Exports:** Verify CSV / Excel exports only extract rows matching the current tenant.
- [x] **Automated Tenant Isolation Tests:** Maintain automated tests asserting cross-tenant queries return 0 rows (tested in `database/tenant_isolation_test.go`).
- [ ] **PostgreSQL Row-Level Security (RLS):** Consider configuring PostgreSQL RLS policies on production tables as defense-in-depth:
  ```sql
  ALTER TABLE reservations ENABLE ROW LEVEL SECURITY;
  CREATE POLICY tenant_isolation_policy ON reservations
      USING (hotel_id = current_setting('app.current_hotel_id')::uuid);
  ```

---

### C. API Security
- [x] **Strict Request Schemas:** Validate all incoming payloads with Go validator tags (`binding:"required,email,min=6"`).
- [x] **SQL Injection Prevention:** Use parameterized prepared statements exclusively via GORM or `pgx`. Never concatenate raw SQL strings.
- [x] **Mass-Assignment Defense:** Use explicit request DTOs instead of binding directly to database models.
- [x] **Public Rate Limiting:** Enforce rate limits on public endpoints (`POST /api/public/bookings`, `POST /api/auth/login`) (e.g. 10 requests / min).
- [x] **Explicit CORS Whitelist:** Configure CORS middleware with explicit trusted domains (`http://localhost:5173`), prohibiting wildcard `*` with credentials.
- [ ] **Enforce HTTPS:** Require TLS 1.3 in production with HSTS (`Strict-Transport-Security`).
- [x] **Request Body Limits:** Limit body size (`http.MaxBytesReader`, e.g., 2 MB for JSON, 10 MB for multipart uploads).
- [ ] **File Upload Validation:** Validate uploaded room photos by checking both MIME type headers and magic bytes (`http.DetectContentType`), restricting extensions to `.jpg`, `.jpeg`, `.png`, `.webp`.
- [x] **Path Traversal Defense:** Sanitize file names; store uploaded assets with generated UUIDs.
- [ ] **SSRF Defense:** Disable or strictly whitelist external URL fetching for images and webhooks.
- [x] **Information Disclosure Defense:** Strip internal database error strings, tracebacks, and server banners from production responses.
- [x] **Consistent JSON Error Structure:** Return unified API errors (`{"success": false, "error": "Human-readable message"}`).
- [x] **IDOR Protection:** Use non-sequential UUIDv4 primary keys across all tables.
- [x] **Pagination Caps:** Require `limit` and `page` on all list endpoints (default 20, max 100).
- [x] **API Versioning:** Prefix routes with `/api/v1/`.

---

### D. Database Security
- [ ] **No Hardcoded Credentials:** Inject database credentials solely via `.env` or Docker secrets.
- [ ] **Network Isolation:** PostgreSQL must listen only on private container networks (never exposed to public internet in production).
- [ ] **Least Privilege Principle:** Production backend user should not have `SUPERUSER` or `DROP TABLE` permissions.
- [ ] **Backup Encryption:** Encrypt database backups with AES-256 before uploading to cold storage.
- [ ] **Encryption at Rest:** Enable storage-level encryption (LUKS / AWS EBS encryption).
- [ ] **TLS DB Connections:** Enforce `sslmode=require` or `verify-full` in staging and production.
- [ ] **Automated Backup & Restore Drill:** Maintain scheduled pg_dump snapshots and test database restoration quarterly.
- [ ] **Foreign Keys & Referential Integrity:** Use explicit FK constraints with `ON DELETE RESTRICT` for financial and room records.
- [ ] **Soft Deletion:** Soft-delete rooms and categories (`gorm.DeletedAt` / `is_deleted = true`) to preserve reservation history.
- [ ] **Audit Logging:** Maintain append-only audit tables for critical modifications (`room_status_history`, `activity_logs`).

---

### E. Payment Security
- [ ] **Zero Card Data Storage:** Never transmit, process, or store raw credit/debit card numbers or CVV codes (PCI-DSS compliance).
- [ ] **Certified Payment Gateways:** Integrate certified gateways (Razorpay, Cashfree, PayU) for card, net banking, and UPI collections.
- [ ] **Backend Signature Verification:** Verify cryptographic HMAC signatures on all payment callbacks before marking reservations paid.
- [ ] **Webhook Signature Verification:** Validate webhook authorization headers and signatures using the gateway webhook secret.
- [ ] **Webhook Idempotency:** Check gateway payment transaction ID before crediting bills to prevent duplicate payment processing:
  ```sql
  SELECT id FROM payments WHERE reference_number = $1;
  ```
- [ ] **Never Trust Frontend Payment Status:** The frontend must never dictate whether a payment was successful; wait for backend webhook or direct gateway server-to-server verification.
- [ ] **Immutable Payment Ledger:** Once a payment record is inserted into the `payments` table, it must never be updated or deleted. Corrections must be handled via credit notes or refund line items.
- [ ] **Refund Audit Trail:** Store refund transaction references, amounts, approved staff user ID, and timestamps.

---

## 2. Performance and Optimization Checklist (High Priority)

### A. Backend Optimization & Go Concurrency
- [ ] **Connection Pooling:** Configure `sql.DB` limits:
  ```go
  sqlDB.SetMaxOpenConns(25)
  sqlDB.SetMaxIdleConns(10)
  sqlDB.SetConnMaxLifetime(1 * time.Hour)
  ```
- [ ] **Eliminate N+1 Queries:** Use GORM `Preload()` or SQL `JOIN` clauses to fetch categories, rooms, and line items in single batches.
- [ ] **Selective Projections:** Use `.Select("id", "name", "rate")` for lightweight list endpoints instead of `SELECT *`.
- [ ] **Asynchronous Background Processing:** Offload emails, SMS notifications, and daily housekeeping rollover to worker queues (e.g., Asynq + Redis).
- [ ] **Goroutine Leak Prevention:** Always pass `context.Context` with timeouts (`context.WithTimeout(ctx, 5*time.Second)`) to prevent stalled goroutines.
- [ ] **Bounded Concurrency:** Never spawn unbounded goroutines per incoming HTTP request; use worker pools.
- [ ] **Memory & Allocation Profiling:** Benchmark hot paths using `pprof` (`go tool pprof http://localhost:8080/debug/pprof/heap`).
- [ ] **Mutex & Lock Minimization:** Avoid coarse mutexes across request handlers; rely on atomic DB operations and Redis locks.
- [ ] **HTTP Server Timeouts:**
  ```go
  server := &http.Server{
      ReadTimeout:  10 * time.Second,
      WriteTimeout: 15 * time.Second,
      IdleTimeout:  60 * time.Second,
  }
  ```
- [ ] **Graceful Shutdown:** Capture `SIGINT` / `SIGTERM` signals and allow active requests 10 seconds to finish before exiting.

---

### B. Database Optimization & Indexing

#### Required Database Indexes:
| Table | Column(s) Indexed | Index Purpose |
| :--- | :--- | :--- |
| **`hotels`** | `status` | Filter active hotel properties |
| **`users`** | `email` [UNIQUE], `status`, `role` | Login lookup & authentication |
| **`hotel_admin_assignments`** | `hotel_id`, `(user_id, hotel_id)` [UNIQUE] | Tenant assignment lookups |
| **`room_categories`** | `hotel_id`, `(hotel_id, name)` [UNIQUE] | Category catalog retrieval |
| **`rooms`** | `(hotel_id, room_number)` [UNIQUE] | Room identity & duplicate prevention |
| **`rooms`** | `hotel_id, status`, `category_id` | Live inventory filtering |
| **`reservations`** | `booking_reference` [UNIQUE] | Public guest reservation lookups |
| **`reservations`** | `hotel_id, check_in, check_out` | Date-window availability queries |
| **`reservations`** | `hotel_id, status` | Front desk operational filters |
| **`bills`** | `hotel_id, status`, `reservation_id` | Invoicing lookups & payment checks |
| **`payments`** | `hotel_id, payment_date`, `bill_id` | Financial reconciliation & ledger |
| **`housekeeping_logs`** | `hotel_id, room_id`, `hk_status` | Staff task assignment |

- [ ] **Overlap Exclusion Constraint (`btree_gist`):**  
  Enforce room reservation exclusivity at the database engine level so two concurrent requests cannot double-book:
  ```sql
  CREATE EXTENSION IF NOT EXISTS btree_gist;
  ALTER TABLE reservations ADD CONSTRAINT no_room_overlap
      EXCLUDE USING gist (
          room_id WITH =,
          daterange(check_in, check_out, '[)') WITH &&
      )
      WHERE (status NOT IN ('cancelled', 'no_show'));
  ```
- [ ] **Query Execution Plans:** Run `EXPLAIN ANALYZE` on critical queries (`availability`, `dashboard stats`) to verify index scans.
- [ ] **Atomic Booking Transactions:** Wrap reservation creation, bill generation, and room status updates inside `db.Transaction(func(tx *gorm.DB) error { ... })`.

---

### C. Caching Strategy (Redis & CDN)

#### Cache Candidates & Strategies:
| Cached Data | Storage Layer | Strategy / TTL | Invalidation Trigger |
| :--- | :--- | :--- | :--- |
| **Hotel Public Profile** | Redis | 1 hour TTL | Hotel update API call |
| **Room Categories & Rates** | Redis | 30 minutes TTL | Category edit / rate change |
| **Hotel Public Images** | CDN (Cloudflare) | 7 days TTL | Image upload / replace |
| **Search Query Filters** | Redis | 15 minutes TTL | Category modifications |
| **Dashboard Statistics** | Redis | 60 seconds TTL | Time expiry or check-in/out |
| **User Role / Permissions** | Redis | 10 minutes TTL | Role edit / user deactivation |
| **Live Room Availability** | Database (Postgres) | **Direct DB Query** | Do not rely on cache for booking |

- [ ] **Tenant-Scoped Cache Keys:** Include tenant ID in every key:
  ```text
  hotel:{hotel_id}:rooms
  hotel:{hotel_id}:categories
  hotel:{hotel_id}:dashboard
  ```
- [ ] **Prevent Cache Stampedes:** Use single-flight suppression or distributed locks when regenerating expensive dashboard metrics.
- [ ] **Zero Caching for Booking Writes:** Never confirm bookings against cached availability; always query PostgreSQL within a transaction.

---

### D. Frontend Optimization
- [ ] **Code Splitting & Lazy Loading:** Use React `lazy()` and `Suspense` for admin dashboards, reporting tabs, and modals.
- [ ] **Modern Image Formats:** Serve room images in `.webp` or `.avif` formats.
- [ ] **Responsive Image Sizing:** Use `srcset` for room cards on mobile vs desktop.
- [ ] **Debounced Search Inputs:** Debounce guest name and room number search fields (300 ms).
- [ ] **Virtualized Lists:** Use virtualized windowing (`react-window`) for reservation tables exceeding 100 rows.
- [ ] **Skeleton Loading:** Display skeleton states during data fetching to maintain perceived performance.
- [ ] **React Render Optimization:** Memoize expensive table calculations with `useMemo` and `useCallback`.
- [ ] **Error Boundaries:** Wrap major feature pages in React Error Boundaries to prevent full-app crashes.

---

### E. Network Optimization
- [ ] **HTTP/2 & HTTP/3:** Enable HTTP/2 multiplexing on the reverse proxy.
- [ ] **Payload Compression:** Enable GZIP / Brotli compression for JSON responses exceeding 1 KB.
- [ ] **HTTP Keep-Alive:** Enable persistent connection reuse between client, proxy, and backend.
- [ ] **Cache-Control Headers:** Send `Cache-Control: public, max-age=31536000` for static assets, and `no-cache, no-store` for authenticated API responses.
- [ ] **Minimize Response Payloads:** Return only necessary fields in public listing APIs.

---

## 3. Hotel-Specific Business Logic Checklist

### A. Booking Management
- [ ] **Multi-Property Group Hierarchy:** Support multiple properties under one hotel group umbrella.
- [ ] **Category-to-Room Mapping:** Support multiple rooms linked to each room category.
- [ ] **Single & Multi-Room Bookings:** Support reserving one or several rooms in a single customer order.
- [ ] **Occupancy Rules:** Validate `adults + children` does not exceed room category `max_occupancy`.
- [ ] **Date Conventions:** `check_in` date is inclusive, `check_out` date is exclusive (`[)`).
- [ ] **Back-to-Back Turnaround:** Allow new guest check-in at 14:00 on the same date a departing guest checks out at 11:00.
- [ ] **Unique Reference Generation:** Format `BK-YYYY-XXXXX` (e.g., `BK-2024-77192`) generated via secure random alphanumeric bytes.
- [ ] **Booking Source Tracking:** Track booking origin: `website`, `walk_in`, `phone`, `ota`.
- [ ] **Reservation State Transitions:**
  ```text
  pending ──► confirmed ──► checked_in ──► checked_out
                  │                │
                  └──► cancelled   └──► (bill closed)
  ```
  * Note: `cancelled` and `checked_out` are terminal states.
- [ ] **Temporary Room Holds:** Support 15-minute checkout holds with TTL expiration while guests complete online payment.

---

### B. Room Management & State Machine
- [ ] **Strict Room Status Definitions:**
  * `AVAILABLE`: Clean, inspected, ready for guest check-in.
  * `RESERVED`: Assigned to an upcoming confirmed reservation.
  * `OCCUPIED`: Guest currently checked in.
  * `DIRTY`: Guest has checked out; requires cleaning.
  * `CLEANING`: Housekeeping actively servicing room.
  * `MAINTENANCE` / `OUT_OF_ORDER`: Physical repairs or plumbing/electrical service.
- [ ] **Decoupled Status Tracking:** Keep physical room status separate from reservation status (a room may be reserved weeks before it becomes occupied).
- [ ] **Room Transfer Workflow:** Allow receptionist to transfer a checked-in guest to another available room with automatic folio transfer and audit logging.

---

### C. Payments and Billing
- [ ] **Automated Folio / Invoice Creation:** Auto-generate bill upon reservation confirmation with room charges.
- [ ] **Multiple Payment Modes:** Support `cash`, `credit_card`, `debit_card`, `upi`, `net_banking`, and `cheque`.
- [ ] **Partial Payments:** Support advance deposits, intermediate payments during long stays, and final check-out settlement.
- [ ] **Service Charges & Extra Folio Items:** Allow staff to post restaurant, laundry, minibar, and spa charges to the guest's folio during their stay.
- [ ] **Discounts & Coupons:** Record percentage or fixed INR discounts with staff authorization note.
- [ ] **Bill Status Lifecycle:**
  * `unpaid` -> `partial` -> `paid` (when balance reaches ₹0.00).
  * `voided` allowed only before any payment has been recorded.
- [ ] **Bill Immutability:** Prohibit editing line items on bills once payments have been applied; adjustments must be issued via credit notes.

---

### D. GST & Indian Localization Engine
- [ ] **Default Currency:** Indian Rupee (`INR` / `₹`).
- [ ] **Numeric Storage:** Store currency amounts using `NUMERIC(12,2)` in PostgreSQL and `float64` / `big.Rat` in Go (never raw floating point).
- [ ] **Configurable GST Policies:** Treat tax as a configurable engine rather than hardcoding 12%:
  * Intra-state supply (Guest & Hotel in same state): 6% CGST + 6% SGST.
  * Inter-state supply (Guest registered in different state): 12% IGST.
- [ ] **Store Tax Entities:**
  * Hotel's GSTIN.
  * Corporate customer's GSTIN (for B2B tax invoice credit).
  * Place of supply (State / Union Territory).
- [ ] **GST Invoice Numbering:** Maintain contiguous sequential financial-year invoice numbers (e.g., `INV/2024-25/0001`).
- [ ] **Credit Notes & Debit Notes:** Support formal credit notes for refunds and disputed charges.
- [ ] **Tax Math Integrity:**
  $$\text{Subtotal} = \sum (\text{item.quantity} \times \text{item.unit\_price})$$
  $$\text{Tax Amount} = \text{round}(\text{Subtotal} \times 0.12, 2)$$
  $$\text{Total Amount} = \text{Subtotal} + \text{Tax Amount} - \text{Discount}$$
  $$\text{Balance} = \text{Total Amount} - \text{Amount Paid}$$

---

### E. Housekeeping & Hotel Services
- [ ] **Task Assignment:** Assign rooms needing service to specific housekeeping staff members.
- [ ] **Status Workflow:** Transition room status from `dirty` -> `cleaning` -> `available` upon completion.
- [ ] **Issue Reporting:** Allow housekeeping to log room maintenance issues (e.g. AC leaking, TV not working).
- [ ] **Facility Charge Posting:** Directly append verified room service, laundry, or minibar usage to `facility_charges`.

---

### F. Guest Management
- [ ] **Guest Directory:** Maintain guest profile repository (`guests` table: name, email, phone, nationality).
- [ ] **Secure Identity Document Storage:** Store ID type (`Aadhaar`, `Passport`, `PAN`, `Driving License`) and masked ID numbers.
- [ ] **Repeat Guest History:** Track past stays, preferences, and booking frequency.
- [ ] **Guest Privacy & Data Retention:** Support customer data deletion requests (DPDP Act compliance) by anonymizing personal identifiable information while preserving tax invoices.

---

## 4. Multi-Tenant Architecture Specification

```
┌─────────────────────────────────────────────────────────────────┐
│               CLIENT APPLICATIONS                               │
│  Guest Booking Portal · Hotel Admin Operations · Super Admin    │
└──────────────────────────────┬──────────────────────────────────┘
                               │ HTTPS / JSON REST
                               │ Authorization: Bearer <JWT>
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│               API GATEWAY / ROUTER (Gin Engine)                 │
│  - CORS & Request Logger Middleware                             │
│  - JWT Verification & Claims Extraction (userID, role, hotelID) │
│  - Tenant Isolation Guard (hotelID required for /api/admin/*)   │
│  - Rate Limiting Middleware                                     │
└──────────────────────────────┬──────────────────────────────────┘
                               │ Context with Tenant ID
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│               DOMAIN SERVICES LAYER                             │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│  │ Auth Service │ │ Booking Svc  │ │ Billing Svc  │             │
│  └──────────────┘ └──────────────┘ └──────────────┘             │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐             │
│  │ Room Service │ │ Housekeeping │ │ Reports Svc  │             │
│  └──────────────┘ └──────────────┘ └──────────────┘             │
└───────────────────────┬───────────────────┬─────────────────────┘
                        │                   │
                        ▼                   ▼
┌────────────────────────────────┐ ┌──────────────────────────────┐
│  PostgreSQL 16 Database        │ │  Redis In-Memory Cache       │
│  - UUID Primary Keys           │ │  - hotel:{id}:categories     │
│  - Tenant Scoped Queries       │ │  - hotel:{id}:rooms          │
│  - btree_gist Exclusion Rules  │ │  - Rate Limiting Buckets     │
└────────────────────────────────┘ └──────────────────────────────┘
```

---

## 5. Testing & Quality Assurance Checklist

### A. Functional Testing
- [ ] Test booking creation by unauthenticated guest.
- [ ] Test room assignment on check-in.
- [ ] Test check-out workflow and automatic status change to `dirty`.
- [ ] Test booking cancellation releasing inventory immediately.
- [ ] Test GST and subtotal calculations across single- and multi-night stays.
- [ ] Test partial payments updating bill balance accurately.
- [ ] Test refund recording and receipt generation.
- [ ] Test concurrent booking attempts on the same room (one succeeds, other gets 409 Conflict).
- [ ] Test room category max occupancy validation.

### B. Security Testing
- [ ] **SQL Injection:** Test raw input in search query parameters (`search=' OR 1=1--`).
- [ ] **XSS:** Test script injection in guest names and special requests fields (`<script>alert(1)</script>`).
- [ ] **IDOR & Cross-Tenant Leakage:** Test Hotel A user querying Hotel B reservation UUID (assert 404/403).
- [ ] **Broken Access Control:** Test `hotel_admin` calling `/api/superadmin/hotels` (assert 403 Forbidden).
- [ ] **Expired Token:** Test API with expired JWT token (assert 401 Unauthorized).
- [ ] **Tampered Token:** Test API with token signed with wrong secret (assert 401 Unauthorized).
- [ ] **Brute Force Defense:** Test 10 rapid incorrect password submissions (assert 429 Too Many Requests).
- [ ] **OWASP Compliance:** Cross-reference against OWASP API Security Top 10 guidelines.

### C. Performance Testing & Benchmarks
- [ ] Benchmark 50 concurrent users.
- [ ] Benchmark 100 concurrent users.
- [ ] Benchmark 500 concurrent users.
- [ ] Benchmark 1,000 concurrent users.
- [ ] Test database connection pool exhaustion handling.
- [ ] Test graceful recovery after database container restart.

### Suggested Performance Targets
| Operation | Engineering Target (P95) |
| :--- | :--- |
| **Cached API request** | P95 < 100 ms |
| **Normal REST API request** | P95 < 300 ms |
| **Database query execution** | P95 < 100 ms |
| **Public hotel & room search** | P95 < 500 ms |
| **Admin dashboard metrics API** | P95 < 500 ms |
| **Booking confirmation transaction** | < 1,000 ms (excluding external payment network) |
| **Frontend Largest Contentful Paint (LCP)** | < 2.5 seconds |
| **Frontend Cumulative Layout Shift (CLS)** | < 0.1 |

---

## 6. DevOps, Deployment & Monitoring Checklist

### A. Infrastructure & CI/CD
- [ ] **Dockerized Services:** Multi-stage `Dockerfile` and `docker-compose.yml` for backend and database.
- [ ] **Environment Separation:** Separate configurations for `development`, `staging`, and `production`.
- [ ] **Nginx Reverse Proxy:** SSL termination, HTTP/2, GZIP, and rate limiting configured.
- [ ] **Zero Secrets in Git:** Validate `.gitignore` blocks `.env`, `*.db`, and private keys.
- [ ] **Automated CI Test Suite:** GitHub Actions workflow running `go test ./...` on every push.
- [ ] **Automated Database Migrations:** Run migrations safely before starting backend container.
- [ ] **Health Checks:** Container healthcheck probing `/health`.
- [ ] **Disaster Recovery:** Scheduled daily PostgreSQL database dumps with retention policy.

### B. Production Monitoring & Telemetry
| Metric Monitored | Warning Threshold | What It Tells You |
| :--- | :--- | :--- |
| **API Latency (P95)** | > 500 ms | Slow endpoints or unindexed database queries |
| **HTTP Error Rate** | > 1% (5xx errors) | Application crashes or unhandled exceptions |
| **CPU Usage** | > 75% sustained | High load, needs scaling or profiling |
| **Memory Usage** | > 80% sustained | Memory leak or unbounded slice allocations |
| **DB Open Connections** | > 20 of 25 conns | Connection pool saturation |
| **Slow Queries** | > 200 ms execution | Missing indexes or table locks |
| **Booking Conflicts** | > 5% failure rate | Concurrency contention or stale availability |
| **Payment Webhook Failures**| Any failure | Payment gateway signature or network drop |

* Recommended Tools: **Prometheus** (metrics), **Grafana** (dashboards), **Sentry** (error reporting), **OpenTelemetry** (distributed tracing).

---

## 7. Overlooked Critical Optimizations

| Optimization | Why It Matters for Hotel Management |
| :--- | :--- |
| **Database Exclusion Constraints** | PostgreSQL `btree_gist` prevents double bookings at the engine level regardless of application thread race conditions |
| **Idempotency Keys** | Header `Idempotency-Key` prevents double charges when network drops during payment requests |
| **Background Job Queues** | Prevents blocking the customer's HTTP request while sending confirmation emails or generating PDF receipts |
| **Redis Distributed Locks** | Coordinates multi-room selection holds across concurrent users |
| **Database Connection Pooling** | Reuses established connections, preventing PostgreSQL process exhaustion |
| **Materialized Views** | Pre-calculates monthly financial aggregations and occupancy percentages for instant dashboard reporting |
| **CDN Image Delivery** | Offloads high-resolution room photos from the API server |
| **Immutable Audit Logging** | Maintains legal evidence of who adjusted rates, authorized discounts, or checked guests in |
| **Circuit Breakers** | Prevents thread pool exhaustion when external SMS/email/payment services experience downtime |

---

## 8. Recommended Development Priorities (Phases 1–6)

```text
Phase 1: Foundation & Security (Completed in Skeleton)
├── Multi-tenant architecture & GORM models
├── JWT authentication & role-based middleware
├── Database auto-migrations & seed data
└── Unit test harness & tenant isolation tests

Phase 2: Core Hotel Operations
├── Hotel and property management APIs
├── Room and room category inventory CRUD
├── Public guest availability & booking engine
└── Check-in, check-out, and room state machine

Phase 3: Financial & Invoicing Engine
├── Bill generation with line items
├── Configurable 12% GST arithmetic (CGST/SGST/IGST)
├── Payment recording & balance reconciliation
└── Razorpay / payment gateway webhook integration

Phase 4: Property Services & Operations
├── Housekeeping task assignment & room turnover
├── Facility charges (laundry, room service, minibar)
└── Guest profile management & booking history

Phase 5: Performance & Optimization
├── Redis caching for hotel profiles & room categories
├── Background worker queues (Asynq)
└── Query profiling with EXPLAIN ANALYZE

Phase 6: Production Readiness & Hardening
├── Automated CI/CD pipeline
├── Penetration & security scanning (OWASP ZAP, Trivy)
├── Load testing with k6 (1,000 concurrent users)
└── Prometheus & Grafana telemetry setup
```

---

## 9. Technology and Tooling Stack

| Layer | Recommended Technology | Current Status |
| :--- | :--- | :--- |
| **Backend Framework** | Go 1.26 + Gin (`github.com/gin-gonic/gin`) | Configured in `Hotel Management` |
| **Database** | PostgreSQL 16 (`uuid-ossp`, `btree_gist`) | Configured in Docker Compose |
| **Database ORM** | GORM (`gorm.io/gorm` + `driver/postgres`) | Configured with AutoMigrate |
| **Caching Layer** | Redis 7 Alpine | Ready for Phase 5 |
| **Frontend Framework**| React 18 + Vite | Configured in `HOTEL-MANGEMENT` |
| **UI Styling** | Tailwind CSS | Configured in frontend |
| **Containerization** | Docker & Docker Compose v5 | Fully configured |
| **Testing Engine** | Go Standard `testing` + `net/http/httptest` | 46/46 tests passing (100% green) |
| **Telemetry & Alerts** | Prometheus + Grafana + Sentry | Ready for Phase 6 |

---

## 10. Master Feature Release Checklist (Mandatory 31-Point Gate)

> **Mandatory PR Rule:** Every developer must complete and verify these 31 checks before merging any new feature or endpoint.

### Security (7 Checks)
- [x] **1. Authentication Checked:** Request requires and validates Bearer JWT token (unless explicitly public).
- [x] **2. Authorization Checked:** Role verified on the backend (`RequireRole`), not relying on frontend UI state.
- [x] **3. Tenant Isolation Verified:** All SQL queries include `WHERE hotel_id = ?` derived from JWT claims.
- [x] **4. Input Validation Implemented:** Request struct validated with strict schema rules (`binding:"required"`).
- [x] **5. SQL Injection & IDOR Defended:** Queries use parameterized statements and UUID keys.
- [x] **6. Sensitive Data Protected:** Passwords, tokens, and customer IDs are omitted from JSON outputs (`json:"-"`).
- [x] **7. Audit Logging Implemented:** Significant operational and financial actions recorded in `activity_logs`.

### Performance (7 Checks)
- [x] **8. Database Queries Optimized:** Query uses an existing index.
- [x] **9. Indexes Reviewed:** Any new high-volume filter column has a composite index.
- [x] **10. N+1 Queries Eliminated:** Related records loaded with GORM `Preload()` or joins.
- [x] **11. Pagination Implemented:** All list endpoints enforce `limit` and `page`.
- [x] **12. Unnecessary Requests Eliminated:** Endpoints return consolidated payloads where practical.
- [x] **13. Caching Evaluated:** Cache evaluated with tenant-scoped keys for read-heavy resources.
- [x] **14. Memory & CPU Overhead Reviewed:** Goroutines are bounded with context timeouts.

### Reliability (6 Checks)
- [x] **15. Error Handling Implemented:** All errors returned via `util.SendError` with proper HTTP codes.
- [x] **16. Database Transactions Used:** Multi-table operations wrapped in `db.Transaction()`.
- [x] **17. Duplicate Requests Handled:** Idempotency checks prevent duplicate bookings or payments.
- [x] **18. Timeouts & Retries Considered:** External calls configured with timeouts.
- [x] **19. Failure Recovery Tested:** System state remains consistent if process restarts mid-workflow.
- [x] **20. Logs & Telemetry Added:** Structured logging captures operational events without sensitive data.

### Testing (6 Checks)
- [x] **21. Unit Tests Written:** Business logic covered in `*_test.go`.
- [x] **22. Integration Tests Written:** HTTP handler endpoints tested with `httptest`.
- [x] **23. Negative Test Cases Covered:** Missing tokens, invalid payloads, and expired sessions tested.
- [x] **24. Cross-Tenant Tests Performed:** Verified Hotel A cannot access Hotel B's records.
- [x] **25. Concurrent Requests Tested:** Overlap prevention tested for simultaneous bookings.
- [x] **26. Documentation Updated:** Endpoint signatures and responses documented in README.

### Deployment & Environment (5 Checks)
- [x] **27. Environment Variables Configured:** New configuration values added to `.env.example`.
- [x] **28. No Secrets Committed:** Verified no keys, tokens, or passwords are committed.
- [x] **29. Database Migrations Tested:** New columns and tables migrate cleanly on fresh database.
- [x] **30. CI/CD Checks Passing:** `go build ./...` and `go test ./...` pass with 0 errors.
- [x] **31. Rollback Plan Available:** Documented reversal steps in case of deployment failure.
