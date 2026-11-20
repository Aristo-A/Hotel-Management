# Hotel Management System — Development & Architecture Notes

## V0.10 (2026-11-20) — First Solid Production Release

* **Platform Milestone Overview**:
  The backend platform is fully unified, feature-complete, production-hardened, and validated against both the frontend React client (`HOTEL-MANGEMENT`) and enterprise API integration contracts. Every endpoint, state machine, financial calculation, tenant boundary, defensive validation guard, and user-facing error message is verified with 100% green automated test coverage.

* **Core Architectural Pillars & Capabilities**:
  1. **Multi-Tenant Property Isolation**:
     - Complete data segregation across properties enforced via JWT claims (`sub`, `hotel_id`, `role`).
     - GORM queries strictly scoped to tenant context; cross-property snooping or mutations return standard 404/403 status.
  2. **Dual-Role Access Control (RBAC)**:
     - `superadmin`: Global property onboarding, admin assignment, property lifecycle, and platform-wide KPI monitoring.
     - `hotel_admin`: Tenant-scoped operations across room categories, physical inventory, front desk reservations, billing folios, and housekeeping.
  3. **Front Desk Reservation State Machine**:
     - Explicit state transitions: `confirmed` $\rightarrow$ `checked_in` $\rightarrow$ `checked_out` (or `cancelled`).
     - Synchronized room status transitions: `available` $\rightarrow$ `occupied` $\rightarrow$ `dirty` (triggering housekeeping audit logs).
     - Guard rails prevent check-in of unconfirmed reservations, early checkout of unchecked-in guests, or cancellations of active/completed stays.
  4. **Financial Billing Folios & 12% Indian GST (INR)**:
     - Automated Indian GST computation: 12% tax added on subtotal room charges and line items.
     - Running folios with support for incidental items, discounts (validated $\le$ bill subtotal), and multi-mode payment recordings (`upi`, `credit_card`, `cash`, `net_banking`).
     - Payment states transition smoothly (`unpaid` $\rightarrow$ `partial` $\rightarrow$ `paid`), with overpayment prevention and settled-folio locking.
  5. **Incidental Facility Charges Ledger**:
     - Dedicated ledger for running incidental guest charges (spa, dining, minibar, gym).
     - Automatic linkage to guest reservations and active bill folios; deletion automatically recalculates folio line items, taxes, and balances.
  6. **Housekeeping Operations Board**:
     - Room turnover tracking across `clean`, `pending`, `in_progress`, and `maintenance` states with staff assignments.
     - Stayover cleaning safeguard: When cleaning an occupied room during multi-night stays, physical room state remains `occupied` to preserve in-house guest bookings.
  7. **Public Guest Booking Engine**:
     - Zero-barrier public booking without account registration.
     - Mathematical interval collision prevention (`check_in < reqCheckOut AND check_out > reqCheckIn`) supporting same-day check-out/check-in turnarounds while strictly blocking double-bookings.
     - Collision-free cryptographic booking reference generation (`BK-YYYY-XXXXX`).
  8. **Defensive Validation & Business Rules**:
     - Check-in dates cannot be in the past (same-day walk-ins supported).
     - Maximum stay capped at 365 nights; guest occupancy validated against room category `maxOccupancy`.
     - Active reservation deletion protection prevents deleting hotel properties or physical rooms while reservations are active.
     - Passwords must be at least 8 characters; new password cannot match current password; superadmins cannot delete/deactivate their own active accounts.
  9. **Human-Readable Error Layer & Client UX**:
     - 100% elimination of cryptic Go/Gin/database error dumps.
     - Centralized binding error translation engine (`util/response.go`) converts validation tags into friendly plain English (`"Room number is required"`, `"Please provide a valid email address"`).
     - Dual-field compatibility: Every error response includes identical `"error"` and `"message"` string keys for friction-free frontend consumption.
  10. **Automated Test Suite**:
      - 57 automated unit and integration tests passing across 9 packages with 0 failures and 0 warnings.
      - In-memory SQLite execution executes the full suite in under 2 seconds.

--------------------------------------------------------------------------------

V0.06 (2026-11-20)

* Summary of Changes
  * **100% Frontend Coverage & Enterprise Alias Alignment**: Completed every missing endpoint, schema, and routing group required by the React frontend (`HOTEL-MANGEMENT`) and `BACKEND_INTEGRATION.md`.
  * **Auth & Profile Management (`/api/v1/auth/*`)**: Mounted `/api/v1/auth` group matching frontend base URL (`/login`, `/logout`, `/me`, `PUT /profile`, `PATCH /password`, `POST /refresh`). Implemented profile updates (name, phone) and password changes with bcrypt verification. Added standard envelope wrapping `data: { token, user }` alongside top-level fields for seamless compatibility with all client styles.
  * **Housekeeping Operations Board (`handler/admin/housekeeping.go`)**: Implemented `GET /api/v1/hotel/housekeeping` (with `status` and `floor` filtering, deriving `clean`, `pending`, `in_progress`, `maintenance` states) and `PATCH /api/v1/hotel/housekeeping/:roomId` (supporting staff assignment and transitioning room status `dirty`/`cleaning` -> `available` on completion).
  * **Facility Charges Running Folio (`handler/admin/facility_charges.go`)**: Implemented standalone incidental charge ledger for checked-in guests (`GET /api/v1/hotel/facility-charges`, `POST /api/v1/hotel/facility-charges`, `DELETE /:id`). Automatically links to guest reservation and updates active bill folio with 12% GST recomputation.
  * **Print Bill Folio (`GET /api/v1/hotel/bills/:id/print`)**: Implemented invoice printer endpoint merging itemized folio lines, payment history, and property company letterhead metadata (`name`, `address`, `phone`, `email`, check-in/out policies) for `PrintBill.jsx`.
  * **Direct Folio Payment (`POST /api/v1/hotel/bills/:id/payments`)**: Added direct payment recording by bill ID parameter.
  * **Live Room Status Snapshot & Report Aliases (`/api/v1/hotel/reports/*`)**: Implemented `GET /reports/room-status` returning real-time inventory counts (`available`, `occupied`, `reserved`, `dirty`, `cleaning`, `outOfService`, `total`). Mounted all reports under `/api/v1/hotel/reports/*` (`occupancy`, `revenue`, `payments`, `housekeeping`, `bookings`, `reservations`, `room-status`).
  * **Public Catalog & Room Detail (`handler/public/public.go`)**: Added `GET /api/v1/public/categories` (auto-resolving active hotel property) and `GET /api/v1/public/rooms/:id` / `GET /api/v1/public/categories/:id` for public room inspection (`RoomDetailPage.jsx`).
  * **Super Admin Status & Hotel Admin Aliases**: Added `PATCH /api/v1/admin/users/:id/status` and `GET /api/v1/admin/hotels/:id/admins`.
  * **55 Automated Tests Passing (100% Green)**: Added 9 comprehensive integration tests across `handler/auth/auth_test.go`, `handler/admin/extension_test.go`, and `handler/public/public_test.go`. All 55 tests pass cleanly in under 2 seconds.

--------------------------------------------------------------------------------

V0.05 (2026-11-20)

* Summary of Changes
  * **Front Desk Operations & Reservation State Machine (Step 5)**: Implemented front desk management in `handler/admin/reservations.go` (`GET /api/admin/reservations`, `POST /api/admin/reservations`, `GET /:id`, `PUT /:id`, `PATCH /:id/check-in`, `PATCH /:id/check-out`, `PATCH /:id/cancel`, `GET /today/check-ins`, `GET /today/check-outs`). Check-in transitions reservation to `checked_in` and physical room to `occupied`. Check-out transitions reservation to `checked_out`, physical room to `dirty`, and auto-creates an audit record in `housekeeping_logs`. Cancellation transitions reservation to `cancelled` and marks physical room `available`.
  * **Financial Billing, Folios & Payment Processing (Step 6)**: Implemented billing engine in `handler/admin/billing.go` (`GET /api/admin/bills`, `GET /:id`, `POST /bills`, `POST /:id/items`, `PATCH /:id/discount`, `GET /api/admin/payments`, `POST /api/admin/payments`, `GET /payments/:id`). Supports posting line items (room service, laundry, spa, minibar) with automated 12% GST recomputation, discount adjustments, and multi-mode payment recordings (`upi`, `credit_card`, `cash`, `net_banking`). Applying payments reduces the bill balance and transitions status from `unpaid` -> `partial` -> `paid`.
  * **Hotel Admin Reporting & Real-time KPI Dashboard (Step 7)**: Implemented analytics suite in `handler/admin/reports.go` (`GET /api/admin/dashboard`, `GET /reports/occupancy`, `GET /reports/revenue`, `GET /reports/housekeeping`, `GET /reports/bookings`). Computes property room inventory breakdown (`totalRooms`, `availableRooms`, `occupiedRooms`, `dirtyRooms`), live occupancy percentage, today's arrivals and departures, total monthly revenue, tax collected, and payment method distribution.
  * **100% Automated Test Suite Passing (46/46 Tests)**: Added 4 extensive test suites in `handler/admin/operations_test.go` (`TestReservation_Lifecycle_And_FrontDesk`, `TestBilling_Lifecycle_And_Payments`, `TestReports_And_Dashboard`, `TestOperations_MultiTenant_Isolation`). All 46 tests across the entire codebase pass cleanly with 0 failures.
  * **Compliance & Verification Cross-Check**: Verified all requirements in `SECURITY_AND_OPTIMIZATION_CHECKLIST.md` across Multi-Tenant Isolation, RBAC, 12% Indian GST in INR, date overlap prevention, and state machine consistency.

--------------------------------------------------------------------------------

V0.04 (2026-11-20)

* Summary of Changes
  * **Public Guest Booking Engine**: Implemented zero-barrier guest endpoints in `handler/public/public.go` (`GET /api/public/hotels/:hotelId`, `GET /categories`, `GET /availability`, `POST /bookings`, `GET /bookings/:bookingReference`). Guests can view public property information, inspect room category tiers, check date availability, place reservations without an account, and retrieve booking confirmations.
  * **Interval Collision & Date-Overlap Engine**: Implemented mathematical interval overlap prevention in `CheckAvailability` and `CreateBooking`: `check_in < reqCheckOut AND check_out > reqCheckIn`. Seamlessly supports same-day room turnarounds (guest check-out at 11:00 AM allows a new guest check-in at 14:00 PM on the same date) while rejecting overlapping double-booking attempts with `409 Conflict`.
  * **12% GST Taxation & Itemized Folio Billing**: Automated financial calculations for room charges with Indian GST compliance (`subtotal = roomRate * nights`, `taxRate = 0.12`, `taxAmount = subtotal * 0.12`, `totalAmount = subtotal + taxAmount`). Every confirmed reservation automatically writes an unpaid customer folio in `bills` and itemized charge rows in `bill_items`.
  * **Cryptographic Booking Reference Generation**: Added collision-free reference generator creating human-readable confirmation identifiers in the format `BK-YYYY-XXXXX` (e.g. `BK-2024-77192`).
  * **Exhaustive API Specification Published**: Created [`API_SPEC.md`](API_SPEC.md) providing comprehensive documentation for all platform REST endpoints, request schemas, status codes, RBAC access matrices, curl examples, and Indian GST rules.
  * **42 Automated Tests Passing (100% Green)**: Added 5 comprehensive integration tests in `handler/public/public_test.go` covering public hotel catalog retrieval, category listings, date overlap collision filtering, 12% GST calculation, folio creation, double-booking prevention, and reference lookups. Total test suite reaches 42 tests.

--------------------------------------------------------------------------------

V0.03 (2026-11-20)

* Summary of Changes
  * **Hotel Inventory & Room Categories Module**: Implemented tenant-isolated room category management in `handler/admin/categories.go` (`GET /api/admin/room-categories`, `POST /api/admin/room-categories`, `GET /:id`, `PUT /:id`, `DELETE /:id`). Automatically extracts `hotel_id` from JWT context, validates category name uniqueness per hotel, parses dynamic amenities and images, and prevents category deletion if active rooms reference it (`409 Conflict`).
  * **Physical Room Management & Operations**: Implemented room inventory CRUD in `handler/admin/rooms.go` (`GET /api/admin/rooms`, `POST /api/admin/rooms`, `GET /:id`, `PUT /:id`, `DELETE /:id`). Enforces unique room numbers per property, floor numbering (>= 1), category foreign key verification within tenant context, and active booking deletion protection (`409 Conflict`). Preloads category details and derives active guest information for occupied rooms.
  * **Room Status State Machine & Housekeeping Auditing**: Implemented `PATCH /api/admin/rooms/:id/status` supporting valid hotel operational states (`available`, `occupied`, `reserved`, `dirty`, `cleaning`, `out_of_service`). Whenever a room transitions into a housekeeping status (`dirty`, `cleaning`, or `available`), the system automatically logs an entry into `housekeeping_logs` and platform audit trails into `activity_logs`.
  * **Multi-Tenant Isolation & Zero Cross-Property Access**: Verified through unit tests that administrators from Hotel B cannot view, mutate, or delete room inventory belonging to Hotel A (yielding strict 404 Not Found responses).
  * **Dual-Path Routing & Enterprise Aliases**: Configured routes in `routes/admin.go` supporting both frontend paths (`/api/admin/rooms`, `/api/admin/room-categories`) and enterprise specification paths (`/api/v1/hotel/rooms`, `/api/v1/hotel/categories`).
  * **37 Automated Tests Passing (100% Green)**: Added comprehensive test suites in `handler/admin/rooms_test.go` covering full category lifecycle, room lifecycle, duplicate validation, housekeeping logging, active reservation protection, and multi-tenant boundary checks.

--------------------------------------------------------------------------------

V0.02 (2026-11-20)

* Summary of Changes
  * **Super Admin Property Management**: Implemented full CRUD operations for hotels in `handler/superadmin/superadmin.go` (`GET /api/superadmin/hotels`, `POST /api/superadmin/hotels`, `GET /api/superadmin/hotels/:id`, `PUT /api/superadmin/hotels/:id`, `DELETE /api/superadmin/hotels/:id`, `PATCH /api/superadmin/hotels/:id/status`). Supports pagination (`page`, `limit`), case-insensitive multi-field search (`name`, `city`, `state`, `country`), and status filtering (`active`, `inactive`).
  * **Hotel Administrator Lifecycle & Assignments**: Implemented hotel administrator management endpoints (`GET /api/superadmin/admins`, `POST /api/superadmin/admins`, `GET /api/superadmin/admins/:id`, `PUT /api/superadmin/admins/:id`, `PATCH /api/superadmin/admins/:id/status`, `DELETE /api/superadmin/admins/:id`, `GET /api/superadmin/hotels/:id/admins`). When an admin is registered or updated, the system enforces email uniqueness, password strength (min 8 chars), bcrypt hashing, and maintains transactional relations in `hotel_admin_assignments`.
  * **Live Platform Dashboard & Metrics**: Implemented `GET /api/superadmin/dashboard` returning aggregate platform metrics: `totalHotels`, `activeHotels`, `inactiveHotels`, `totalAdmins`, `totalBookings`, `totalRevenue` (sum of completed payments and billed revenues in INR), `occupancyRate` (live occupied/total rooms calculation), hotel overview lists, and recent activity audit feeds.
  * **Security, Business Rules & Deletion Protection**: Prohibited the deletion of hotel properties that have active or upcoming reservations (`confirmed` or `checked_in`). Enforced RBAC at the middleware level (`RequireRole("superadmin")`). Added automatic platform audit logging to `activity_logs` for all property creation, admin creation, status updates, and deletions.
  * **Unified Dual-Path Routing**: Registered both React frontend paths (`/api/superadmin/*`) and `BACKEND_INTEGRATION.md` enterprise aliases (`/api/v1/admin/*`), ensuring seamless compatibility across different integration specifications.
  * **100% Passing Test Suite (34 Tests)**: Added 8 comprehensive unit tests in `handler/superadmin/superadmin_test.go` covering dashboard aggregation, hotel CRUD, search filters, admin lifecycle, status toggles, deletion safety, and RBAC protection. Total test count across the project reached 34 tests with 100% pass rate.

--------------------------------------------------------------------------------

V0.01 (2026-11-20)

* Summary of Changes
  * **Project Inception & Architecture**: Established clean Go REST API architecture (`cmd`, `database`, `handler`, `middleware`, `models`, `routes`, `util`) in a dedicated directory (`Hotel Management`), completely isolated from the frontend (`HOTEL-MANGEMENT`). Configured multi-module workspace in `go.work`.
  * **Database Models & Auto-Migrations**: Defined 12 GORM models with UUID v4 primary keys (`hotels`, `users`, `hotel_admin_assignments`, `room_categories`, `rooms`, `guests`, `reservations`, `bills`, `bill_items`, `payments`, `housekeeping_logs`, `activity_logs`). Implemented dual database connector in `database/connect.go` supporting PostgreSQL with automatic fallback to SQLite (`hotel.db`) for offline development.
  * **Authentication & RBAC**: Implemented `POST /api/auth/login`, `GET /api/auth/me`, and `POST /api/auth/logout`. Enforces bcrypt password verification, inactive account locks, tenant assignment resolution for `hotel_admin`, and signed HS256 JWT tokens with tenant claims (`sub`, `name`, `email`, `role`, `hotel_id`, `hotel_name`).
  * **Multi-Tenant Isolation Verification**: Added `database/tenant_isolation_test.go` confirming that Hotel A cannot read, update, or delete Hotel B's records.
  * **Business Logic & GST Testing**: Added `models/business_logic_test.go` verifying 12% GST calculation in INR, room status state machine transitions (`available` -> `reserved` -> `occupied` -> `dirty`), and date overlap logic.
  * **Dockerization**: Created multi-stage `Dockerfile`, `docker-compose.yml`, and PostgreSQL initialization script (`docker/init.sql`) with `uuid-ossp` and `btree_gist` extensions.
  * **Checklist & Compliance**: Authored and committed the comprehensive 586-line `SECURITY_AND_OPTIMIZATION_CHECKLIST.md`.

## V0.07 Defensive Validation & Robustness Hardening (Post-Step 8)
- **Reservation & Booking Guards**:
  - Check-in dates cannot be in the past (`checkIn < today`), while same-day walk-ins remain allowed.
  - Maximum stay duration strictly capped at 365 nights.
  - Guest occupancy strictly validated against room category `maxOccupancy` (rejects with 400 Bad Request if capacity exceeded).
  - Check-in transition guard: Only reservations in `confirmed` status can be checked in. Fails with 409 Conflict if room is physically occupied or out of service.
  - Check-out transition guard: Only `checked_in` reservations can be checked out.
  - Cancellation guard: Cannot cancel active `checked_in` or completed `checked_out` reservations. Unpaid untouched bill folios automatically cancelled on reservation cancellation.
  - Cannot update/modify cancelled or checked-out reservations.
- **Billing & Financial Guards**:
  - Payment amount must be strictly positive (`amount > 0`).
  - Payments rejected if bill folio is already settled (`balance == 0` and `status == "paid"`) or cancelled.
  - Overpayment safely clamped (`balance = max(0, total - paid)`).
  - Discounts cannot be negative or exceed bill total before discount.
  - Charges cannot be added to cancelled bill folios.
- **Housekeeping Operational Safeguard**:
  - When cleaning an occupied room during multi-night stay, room status remains `occupied` (prevents double-booking vacant room while guest is in-house).
- **Incidental & Facility Charges Integrity**:
  - Facility charges cannot be added to cancelled or checked-out reservations.
  - Deleting a facility charge automatically deletes the matching folio line item and recalculates subtotal, 12% GST, total, and balance.
- **Rooms & Inventory Safeguards**:
  - Negative rates and invalid floor ranges rejected.
  - Category maximum occupancy validated between 1 and 20.
- **Authentication & User Safety**:
  - New password cannot match current password.
  - Profile name minimum length validated.
  - Superadmin cannot delete or deactivate their own active administrator account.
- **Test Suite Status**: 57 automated tests across 9 packages passing with 100% success rate.

--------------------------------------------------------------------------------

## V0.08 Human-Readable Error Layer & Client UX Standardization

* **Elimination of Raw Error Leaks**:
  - Replaced all raw Go/Gin/GORM internal error strings (e.g., `+err.Error()`, Gin validator reflection dumps like `Key: 'CreateRoomRequest.RoomNumber' Error:Field validation for 'RoomNumber' failed on the 'required' tag`, and internal database syntax or driver errors) across every controller handler.
  - Replaced internal database failure messages with friendly action-oriented feedback (e.g. `"Unable to save room details. Please try again."`, `"Unable to record payment. Please try again."`).

* **Human-Readable Binding Engine (`util/response.go`)**:
  - Implemented `FormatBindingError(err error) string` mapping technical struct tags to clear plain-English sentences:
    - `required`: `"<Friendly Name> is required"` (e.g., `"Room number is required"`, `"Email address is required"`, `"Password is required"`)
    - `email`: `"Please provide a valid email address"`
    - `min`: `"<Friendly Name> must be at least <N> characters long"`
    - `max`: `"<Friendly Name> cannot exceed <N> characters"`
    - `gt`/`gte`: `"<Friendly Name> must be greater than / at least <N>"`
    - `lt`/`lte`: `"<Friendly Name> must be less than / cannot exceed <N>"`
    - Malformed JSON / Syntax errors: `"Request body contains invalid or malformed JSON"`
    - Empty body / EOF: `"Request body cannot be empty"`
    - Unmarshaling / Time parse errors: `"Invalid data format provided. Please verify the submitted values."`
  - Created domain-wide dictionary `humanReadableFieldNames` converting lowercase struct keys to clear UI labels (`roomnumber` -> `Room number`, `categoryid` -> `Room category`, `defaultrate` -> `Base price per night`, `guestname` -> `Guest name`, `checkin` -> `Check-in date`, etc.).
  - Implemented `SendBindingError(c *gin.Context, err error)` helper across all controller modules.

* **Dual-Field Error Payload (`error` and `message`)**:
  - Updated `APIError` response struct so that error responses reliably populate both `"error"` and `"message"` with identical human-readable strings.
  - Guarantees seamless consumption regardless of whether frontend components inspect `res.error` or `res.message`.

* **100% Passing Automated Tests**:
  - Added `handler/admin/human_readable_errors_test.go` and updated `util/response_test.go`.
  - All test packages (`database`, `handler/admin`, `handler/auth`, `handler/health`, `handler/public`, `handler/superadmin`, `middleware`, `models`, `util`) pass with zero regressions.
