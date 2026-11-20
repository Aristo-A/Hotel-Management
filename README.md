# Hotel Management System — Go Backend Platform (v0.10)

![Version](https://img.shields.io/badge/version-v0.10-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)
![Tests](https://img.shields.io/badge/tests-57%20passing-brightgreen.svg)
![Status](https://img.shields.io/badge/status-production--ready-success.svg)

An enterprise-grade, multi-tenant Hotel Management System backend built with Go (Gin), PostgreSQL / SQLite, GORM, and JWT. It provides complete tenant isolation, role-based access control (RBAC), 12% GST compliance in Indian Rupees (₹), robust front desk reservation management, running billing folios, and standardized human-readable validation.

Modeled after clean, production-hardened monolithic Go architecture and completely integrated with the React frontend in `HOTEL-MANGEMENT`.

---

## 🚀 Features

*   **Multi-Tenant Architecture**: Strict property isolation where hotel administrators and staff access only their assigned property's records (`WHERE hotel_id = $hotel_id_from_jwt`).
*   **Dual-Role Access Control (RBAC)**: Support for platform `superadmin` and property-level `hotel_admin`.
*   **Property & Administrator Management**: SuperAdmin can register hotel properties, manage operations, toggle statuses, and create/assign property managers.
*   **Hotel Room & Category Inventory**: Full CRUD for room tiers (`Deluxe`, `Suite`, `Standard`) and physical rooms with room status state machine and housekeeping logging.
*   **Real-time Platform Dashboard**: Aggregate platform-wide statistics (`totalHotels`, `activeHotels`, `totalAdmins`, `totalRevenue`, `occupancyRate`, `totalBookings`, `recentActivities`).
*   **Enterprise Security**: Bcrypt password hashing, session tokens, login rate limiting, action auditing, and deletion protection for properties or rooms with active guest bookings.
*   **GST & Indian Currency Compliance**: Built-in 12% GST taxation calculations and INR currency handling.
*   **Dual Database Support**: Primary PostgreSQL 16 connection pool with automatic fallback to embedded SQLite (`hotel.db`) for instant offline local testing.
*   **Dockerized Deployment**: Production-ready multi-stage Alpine Dockerfile and Docker Compose orchestration.

---

## 🛠️ Prerequisites & Setup

### Option 1: Docker Compose (Recommended)

Run the backend and PostgreSQL database together:
```bash
docker-compose up --build
```
*Note: On first startup, the database auto-migrates all 12 core tables and seeds initial demo Super Admin (`superadmin@hotelgroup.com` / `SecurePassword123!`) and Hotel Admin (`admin@grandhorizon.com` / `SecurePassword123!`) accounts.*

### Option 2: Local Go Development

1. Ensure Go 1.21+ is installed.
2. In this directory (`Hotel Management`):
```bash
go mod tidy
go run cmd/server/main.go
```
The server will start at `http://localhost:8080`.
Verify system health at `http://localhost:8080/health`.

### Run Unit Tests
```bash
go test -v ./...
```
*(All 55 automated unit and integration tests run in-memory and execute in under 2 seconds).*

---

## 📚 API Documentation (For Frontend Teams)

Base URL: `http://localhost:8080`

### 1. Authentication Endpoints

| Method | Endpoint | Description | Payload Example / Notes |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/auth/login` | Authenticate superadmin or hotel admin | `{"email": "admin@grandhorizon.com", "password": "password123"}` |
| `GET` | `/api/auth/me` | Fetch authenticated user profile & tenant context | Requires `Authorization: Bearer <JWT>` |
| `POST` | `/api/auth/logout` | Session invalidation confirmation | Requires `Authorization: Bearer <JWT>` |
| `PUT` | `/api/v1/auth/profile` | Update profile details (name, phone) | `{"name": "Arjun Sharma", "phone": "+91 98111 22222"}` |
| `PATCH`| `/api/v1/auth/password`| Change authenticated user password | `{"currentPassword": "...", "newPassword": "..."}` |
| `POST` | `/api/v1/auth/refresh` | Extend access token validity | Requires `Authorization: Bearer <JWT>` |

---

### 2. SuperAdmin Endpoints (Requires `Authorization: Bearer <SuperAdmin_JWT>`)

*(Also available under enterprise alias `/api/v1/admin/*`)*

| Method | Endpoint | Description | Payload Example |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/superadmin/dashboard` | Fetch global platform metrics, recent activities & hotel overview | None |
| `GET` | `/api/superadmin/hotels` | List all hotels with room counts, assigned admins & pagination | Query: `?page=1&limit=10&search=mumbai&status=active` |
| `POST` | `/api/superadmin/hotels` | Register a new hotel property and optionally link an admin | `{"name": "Seaside Resort", "description": "Beachfront property", "address": "Calangute Road", "city": "Goa", "phone": "+91 832 227 1234", "email": "stay@seaside.in", "starRating": 5, "adminId": "user-uuid"}` |
| `GET` | `/api/superadmin/hotels/:id` | Retrieve single hotel property details with room counts | None |
| `PUT` | `/api/superadmin/hotels/:id` | Update hotel property details or reassign admin | Same payload as POST |
| `PATCH` | `/api/superadmin/hotels/:id/status`| Toggle or set hotel status (`active` / `inactive`) | `{"status": "inactive"}` (or empty body to toggle) |
| `DELETE` | `/api/superadmin/hotels/:id` | Delete hotel (blocked if active reservations exist) | None |
| `GET` | `/api/superadmin/hotels/:id/admins`| List all administrators assigned to a specific hotel | None |
| `GET` | `/api/superadmin/admins` | List all hotel administrators with assigned property info | Query: `?page=1&limit=10&search=arjun&status=active` |
| `POST` | `/api/superadmin/admins` | Create new hotel administrator account and assign to hotel | `{"name": "Priya Nair", "email": "priya@seaside.in", "phone": "+91 98333 44444", "password": "SecurePassword123!", "hotelId": "hotel-uuid", "status": "active"}` |
| `GET` | `/api/superadmin/admins/:id` | Retrieve single administrator details | None |
| `PUT` | `/api/superadmin/admins/:id` | Update administrator details or reassign hotel | `{"name": "Priya N. Sharma", "phone": "+91 98333 55555", "hotelId": "hotel-uuid"}` |
| `PATCH` | `/api/superadmin/admins/:id/status`| Toggle administrator account status (`active` / `inactive`)| `{"status": "inactive"}` (or empty body to toggle) |
| `DELETE` | `/api/superadmin/admins/:id` | Delete administrator account | None |

---

### 3. Hotel Admin Inventory Endpoints (Requires `Authorization: Bearer <HotelAdmin_JWT>`)

*(Also available under enterprise alias `/api/v1/hotel/*`)*

#### Room Categories (`/api/admin/room-categories`)
| Method | Endpoint | Description | Payload Example |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/admin/room-categories` | List all room categories for this hotel with room counts | None |
| `POST` | `/api/admin/room-categories` | Create a new room tier (unique name per hotel) | `{"name": "Deluxe Suite", "description": "Sea view suite", "defaultRate": 5500, "maxOccupancy": 2, "bedType": "King", "roomSize": 38, "amenities": ["WiFi", "AC"], "image": "https://..."}` |
| `GET` | `/api/admin/room-categories/:id` | Retrieve single category details | None |
| `PUT` | `/api/admin/room-categories/:id` | Update category attributes or base rate | Same payload as POST |
| `DELETE` | `/api/admin/room-categories/:id` | Delete category (rejected with `409 Conflict` if rooms exist) | None |

#### Physical Rooms (`/api/admin/rooms`)
| Method | Endpoint | Description | Payload Example / Notes |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/admin/rooms` | List rooms filtered by status, floor, search, or category | Query: `?status=available&floor=1&search=101&categoryId=uuid` |
| `POST` | `/api/admin/rooms` | Add a new physical room to property inventory | `{"roomNumber": "101", "floor": 1, "categoryId": "cat-uuid", "rate": 5500, "status": "available", "notes": "Sea view"}` |
| `GET` | `/api/admin/rooms/:id` | Get room detail with preloaded category and current guest | None |
| `PUT` | `/api/admin/rooms/:id` | Update room number, floor, rate, or category | Same payload as POST |
| `PATCH` | `/api/admin/rooms/:id/status` | Update room status (`available`, `occupied`, `reserved`, `dirty`, `cleaning`, `out_of_service`) and log housekeeping | `{"status": "cleaning", "notes": "Deep sanitization", "assignedTo": "Maria"}` |
| `DELETE` | `/api/admin/rooms/:id` | Delete room (rejected with `409 Conflict` if active booking exists) | None |

#### Housekeeping Board (`/api/admin/housekeeping` & `/api/v1/hotel/housekeeping`)
| Method | Endpoint | Description | Payload Example / Notes |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/hotel/housekeeping` | List room turnover tasks and staff assignments | Query: `?status=pending&floor=1` |
| `PATCH` | `/api/v1/hotel/housekeeping/:roomId` | Update cleaning status (`clean`, `in_progress`, `pending`, `maintenance`) | `{"taskStatus": "completed", "assignedTo": "Meena", "notes": "Sanitized"}` |

---

### 4. Public Guest Endpoints (No Auth Required)

*(Also available under enterprise alias `/api/v1/public/*`)*

| Method | Endpoint | Description | Payload Example / Notes |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/public/hotels/:hotelId` | Public hotel profile, amenities & check-in policies | Alias: `/api/v1/public/hotel` |
| `GET` | `/api/v1/public/categories` | Public catalog of room category tiers | Default active hotel resolved |
| `GET` | `/api/v1/public/rooms/:id` | Public room & category detail with available room | Alias: `/api/v1/public/categories/:id` |
| `GET` | `/api/public/hotels/:hotelId/availability` | Query available rooms with date-overlap prevention | Query: `?checkIn=2024-12-01&checkOut=2024-12-04&guests=2` |
| `POST` | `/api/public/bookings` | Zero-barrier guest booking (auto-calculates 12% GST in INR and creates folio) | `{"hotelId": "uuid", "roomId": "uuid", "guestName": "Priya Sharma", "guestEmail": "priya@example.com", "guestPhone": "+91 98700 12345", "checkIn": "2024-12-01", "checkOut": "2024-12-04", "numGuests": 2}` |
| `GET` | `/api/public/bookings/:bookingReference` | Guest confirmation lookup by booking reference code | None |

---

### 5. Front Desk Reservations & Room Operations (Requires `Authorization: Bearer <HotelAdmin_JWT>`)

*(Also available under enterprise alias `/api/v1/hotel/reservations`)*

| Method | Endpoint | Description | Payload Example / Notes |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/admin/reservations` | List reservations with filters (`status`, `search`, `roomId`, `checkIn`, `checkOut`) | Query: `?status=confirmed&search=priya&page=1&limit=20` |
| `POST` | `/api/admin/reservations` | Front desk / walk-in booking creation | `{"roomId": "uuid", "guestName": "Rahul Verma", "guestEmail": "rahul@example.com", "guestPhone": "+91 98111 22334", "checkIn": "2024-12-15", "checkOut": "2024-12-18", "numGuests": 2}` |
| `GET` | `/api/admin/reservations/:id` | Retrieve reservation with room, category, guest & folio details | None |
| `PUT` | `/api/admin/reservations/:id` | Update reservation details | `{"roomId": "new-uuid", "numGuests": 3}` |
| `PATCH` | `/api/admin/reservations/:id/check-in` | Guest check-in (transitions reservation -> `checked_in`, room -> `occupied`) | None |
| `PATCH` | `/api/admin/reservations/:id/check-out` | Guest check-out (transitions reservation -> `checked_out`, room -> `dirty`, logs housekeeping) | None |
| `PATCH` | `/api/admin/reservations/:id/cancel` | Cancel reservation (transitions reservation -> `cancelled`, room -> `available`) | `{"reason": "Guest cancellation"}` |
| `GET` | `/api/admin/reservations/today/check-ins` | Today's expected guest arrivals | None |
| `GET` | `/api/admin/reservations/today/check-outs` | Today's expected guest departures | None |

---

### 6. Billing, Invoicing & Payments (Requires `Authorization: Bearer <HotelAdmin_JWT>`)

*(Also available under enterprise alias `/api/v1/hotel/bills` and `/api/v1/hotel/payments`)*

| Method | Endpoint | Description | Payload Example / Notes |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/admin/bills` | List customer folios/invoices with status & search filters | Query: `?status=partial&search=INV-2024` |
| `GET` | `/api/admin/bills/:id` | Get bill details with line items, payments & guest details | None |
| `POST` | `/api/admin/bills/:id/items` | Add facility charge (room service, laundry, spa, minibar) with auto 12% GST calculation | `{"itemType": "restaurant", "description": "Room Service Dinner", "quantity": 2, "unitPrice": 600}` |
| `PATCH` | `/api/admin/bills/:id/discount` | Apply discount to bill (recalculates total & balance) | `{"discount": 1000}` |
| `GET` | `/api/admin/payments` | List payment ledger entries | Query: `?billId=uuid&page=1` |
| `POST` | `/api/admin/payments` | Record payment (auto-reduces balance, transitions bill status to `paid` or `partial`) | `{"billId": "uuid", "amount": 11800, "paymentMethod": "upi", "transactionId": "UPI-123456"}` |
| `POST` | `/api/v1/hotel/bills/:id/payments` | Record payment directly against folio | Same as POST /payments |
| `GET` | `/api/v1/hotel/bills/:id/print` | Printable bill with letterhead | Complete invoice with hotel header |
| `GET` | `/api/v1/hotel/facility-charges` | List running facility charges for folio | Query: `?reservationId=uuid&category=spa` |
| `POST` | `/api/v1/hotel/facility-charges` | Add facility charge (restaurant, spa, minibar) | `{"reservationId": "uuid", "category": "spa", "description": "Massage", "amount": 2500}` |
| `DELETE` | `/api/v1/hotel/facility-charges/:id` | Remove erroneous facility charge | None |
| `GET` | `/api/admin/payments/:id` | Retrieve single payment receipt | None |

---

### 7. Hotel Admin Reporting & Analytics (Requires `Authorization: Bearer <HotelAdmin_JWT>`)

*(Also available under enterprise alias `/api/v1/hotel/reports/*`)*

| Method | Endpoint | Description | Notes |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/admin/dashboard` | Hotel operations snapshot (room statuses, arrivals, departures, monthly revenue, occupancy rate) | Instant operational KPI view |
| `GET` | `/api/admin/reports/occupancy` | Occupancy breakdown by category and property-wide percentage | Live inventory metrics |
| `GET` | `/api/admin/reports/revenue` | Revenue report with GST collected, discounts, outstanding balances, and payment methods | Query: `?startDate=2024-12-01&endDate=2024-12-31` |
| `GET` | `/api/admin/reports/housekeeping` | Housekeeping history, room turnover status & cleaning logs | Operational audit trail |
| `GET` | `/api/admin/reports/bookings` | Booking distribution by status and arrival timelines | Reservation analytics |

> 📖 **Full API Specifications & Schema Documentation:** See [`API_SPEC.md`](API_SPEC.md) for full request/response JSON schemas, error codes, and curl examples across all endpoints.

---

### 🛡️ Standardized Error Response Envelope

All error responses across every endpoint return human-readable messages with dual-field compatibility (`error` and `message`):

```json
{
  "success": false,
  "error": "Room number is required",
  "message": "Room number is required"
}
```

Zero raw Go/Gin validator reflection paths (e.g. `Field validation for 'RoomNumber' failed`) or database internal driver errors are leaked to the client.

---

## 📋 Implementation Roadmap

- [x] **Step 1: Auth Module & Multi-Tenant Foundations** (`POST /api/auth/login`, `GET /me`, `POST /logout`, multi-tenant models & tests)
- [x] **Step 2: Super Admin Property Operations & Dashboard** (Hotels CRUD, Admins CRUD, status toggling, global metrics, activity auditing)
- [x] **Step 3: Hotel Inventory & Rooms** (`/api/admin/room-categories`, `/api/admin/rooms`, room status state machine, housekeeping logging)
- [x] **Step 4: Public Guest Engine** (`/api/public/hotels/:id`, `/availability`, zero-barrier `/bookings`, reservation reference lookup, 12% GST billing)
- [x] **Step 5: Front Desk Reservations & Check-in/Check-out** (Reservation search, check-in, check-out with room state transitions)
- [x] **Step 6: Billing, Invoicing & Payments** (12% GST calculation, line items, payment processing, invoice generation)
- [x] **Step 7: Hotel Admin Reporting & Dashboards** (Occupancy, revenue, housekeeping, booking trends)
- [x] **Step 8: 100% Frontend Integration & Enterprise Aliases** (Housekeeping operational board, facility charges running folios, printable letterhead folios, profile management, public catalog)
- [x] **Step 9: Defensive Validation & Integrity Hardening** (Capacity limits, date interval overlap prevention, past date rejection, status transition locks, settled-bill safety)
- [x] **Step 10: Human-Readable Error Layer (v0.10)** (Centralized binding translation, dual envelope, friendly plain-English messaging across 100% of endpoints)
