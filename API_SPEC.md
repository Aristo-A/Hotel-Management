# Hotel Management System — Comprehensive API Specifications

**Base URL (Local):** `http://localhost:8080`  
**Base URL (Production):** `https://api.yourhotelportal.com`  
**Currency:** Indian Rupee (`INR` / `₹`)  
**Taxation Standard:** 12% GST (`taxRate = 0.12`)  
**Specification Version:** `1.0.0`

---

## 🔐 Authentication & Global Headers

All secured requests require a valid JSON Web Token (JWT) provided in the HTTP `Authorization` header:

```http
Authorization: Bearer <ACCESS_TOKEN>
Content-Type: application/json
Accept: application/json
```

### Standard Roles & RBAC Matrix
| Role | Permissions & Context |
| :--- | :--- |
| `superadmin` | Global platform administration. Can access all hotels, assign managers, and view aggregated cross-property revenue. |
| `hotel_admin` | Property-level manager. Scoped strictly to assigned `hotel_id` from the JWT claims (`WHERE hotel_id = $claims.hotel_id`). |
| *Guest / Public* | Zero-barrier public booking endpoints (`/api/public/*`). No authentication required. |

### Standard Response Envelope
All API endpoints follow a unified response structure:

#### Success (200 OK / 201 Created)
```json
{
  "success": true,
  "message": "Optional human-readable confirmation message",
  "data": { ... },
  "meta": {
    "total": 100,
    "page": 1,
    "limit": 10
  }
}
```

#### Error (400, 401, 403, 404, 409, 422, 500)
```json
{
  "success": false,
  "error": "Human-readable description of the error",
  "message": "Human-readable description of the error",
  "details": null
}
```

---

## 1. Authentication Endpoints (`/api/auth`)

### 1.1 Login
* **Method:** `POST`
* **Path:** `/api/auth/login`
* **Access:** Public
* **Description:** Authenticates a superadmin or hotel admin, returning a stateless JWT token and user profile.
* **Request Body:**
```json
{
  "email": "admin@grandhorizon.com",
  "password": "SecurePassword123!"
}
```
* **Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Authentication successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "c1f7a0b9-52e8-466d-a3df-60e1d842d0a1",
      "name": "Arjun Sharma",
      "email": "admin@grandhorizon.com",
      "role": "hotel_admin",
      "hotelId": "d0e1b2c3-4567-89ab-cdef-0123456789ab",
      "hotelName": "Grand Horizon Hotel"
    }
  }
}
```
* **Error Responses:**
  * `401 Unauthorized`: Invalid email or password.
  * `403 Forbidden`: Account deactivated by administrator.

### 1.2 Get Authenticated User Context
* **Method:** `GET`
* **Path:** `/api/auth/me`
* **Access:** Authenticated (`superadmin`, `hotel_admin`)
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "c1f7a0b9-52e8-466d-a3df-60e1d842d0a1",
    "name": "Arjun Sharma",
    "email": "admin@grandhorizon.com",
    "role": "hotel_admin",
    "hotelId": "d0e1b2c3-4567-89ab-cdef-0123456789ab",
    "hotelName": "Grand Horizon Hotel"
  }
}
```

### 1.3 Logout
* **Method:** `POST`
* **Path:** `/api/auth/logout` (Alias: `/api/v1/auth/logout`)
* **Access:** Authenticated
* **Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Successfully logged out"
}
```

### 1.4 Update Authenticated User Profile
* **Method:** `PUT`
* **Path:** `/api/auth/profile` (Alias: `/api/v1/auth/profile`)
* **Access:** Authenticated
* **Request Body:**
```json
{
  "name": "Arjun Sharma",
  "phone": "+91 98111 22222"
}
```
* **Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Profile updated successfully",
  "data": {
    "id": "c1f7a0b9-52e8-466d-a3df-60e1d842d0a1",
    "name": "Arjun Sharma",
    "email": "admin@grandhorizon.com",
    "role": "hotel_admin",
    "phone": "+91 98111 22222",
    "hotelId": "d0e1b2c3-4567-89ab-cdef-0123456789ab",
    "hotelName": "Grand Horizon Hotel"
  }
}
```

### 1.5 Change Password
* **Method:** `PATCH` or `PUT`
* **Path:** `/api/auth/password` (Alias: `/api/v1/auth/password`)
* **Access:** Authenticated
* **Request Body:**
```json
{
  "currentPassword": "SecurePassword123!",
  "newPassword": "NewStrongPassword456!"
}
```
* **Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Password changed successfully"
}
```

### 1.6 Refresh Access Token
* **Method:** `POST`
* **Path:** `/api/auth/refresh` (Alias: `/api/v1/auth/refresh`)
* **Access:** Authenticated
* **Success Response (200 OK):**
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

---

## 2. Super Admin Endpoints (`/api/superadmin` & `/api/v1/admin`)

*Required Role:* `superadmin`

### 2.1 Platform Dashboard
* **Method:** `GET`
* **Path:** `/api/superadmin/dashboard` (Alias: `/api/v1/admin/stats`)
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "totalHotels": 3,
    "activeHotels": 3,
    "inactiveHotels": 0,
    "totalAdmins": 4,
    "totalRevenue": 28500000.00,
    "occupancyRate": 72.4,
    "totalBookings": 1240,
    "currency": "INR",
    "hotels": [
      {
        "id": "d0e1b2c3-4567-89ab-cdef-0123456789ab",
        "name": "Grand Horizon Hotel",
        "city": "Mumbai",
        "totalRooms": 120,
        "availableRooms": 45,
        "adminName": "Arjun Sharma",
        "status": "active"
      }
    ],
    "recentActivities": [
      {
        "id": "act-1",
        "type": "create_public_booking",
        "description": "Guest Priya Sharma confirmed reservation BK-2024-77192",
        "time": "2024-11-20T10:30:00Z",
        "user": "System"
      }
    ]
  }
}
```

### 2.2 List Hotels
* **Method:** `GET`
* **Path:** `/api/superadmin/hotels` (Alias: `/api/v1/admin/hotels`)
* **Query Parameters:**
  * `page` (int, default: 1)
  * `limit` (int, default: 10)
  * `search` (string): Filters across name, city, state, country.
  * `status` (string): `all`, `active`, `inactive`.
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "d0e1b2c3-4567-89ab-cdef-0123456789ab",
      "name": "Grand Horizon Hotel",
      "description": "Luxury 5-star seafront hotel",
      "address": "Marine Drive, Nariman Point",
      "city": "Mumbai",
      "state": "Maharashtra",
      "country": "India",
      "phone": "+91 22 6610 8888",
      "email": "contact@grandhorizon.com",
      "image": "https://images.unsplash.com/photo-1566073771259-6a8506099945",
      "status": "active",
      "starRating": 5,
      "rating": 4.8,
      "totalRooms": 120,
      "availableRooms": 45,
      "adminName": "Arjun Sharma",
      "adminEmail": "admin@grandhorizon.com",
      "createdAt": "2024-01-15T08:00:00Z"
    }
  ],
  "meta": { "total": 1, "page": 1, "limit": 10 }
}
```

### 2.3 Create Hotel Property
* **Method:** `POST`
* **Path:** `/api/superadmin/hotels` (Alias: `/api/v1/admin/hotels`)
* **Request Body:**
```json
{
  "name": "Seaside Paradise Resort",
  "description": "Luxury beachfront property in North Goa",
  "address": "Calangute Beach Road, 403516",
  "city": "Goa",
  "state": "Goa",
  "country": "India",
  "phone": "+91 832 227 1234",
  "email": "stay@seasideparadise.in",
  "website": "https://seasideparadise.in",
  "image": "https://images.unsplash.com/photo-1520250497591-112f2f40a3f4",
  "starRating": 5,
  "status": "active",
  "checkInTime": "14:00",
  "checkOutTime": "11:00",
  "adminId": "user-uuid"
}
```
* **Success Response (201 Created):** Returns the created hotel object with generated UUID.

### 2.4 Update Hotel Details
* **Method:** `PUT`
* **Path:** `/api/superadmin/hotels/:id`
* **Request Body:** Same schema as `POST /hotels`. Partial updates supported.

### 2.5 Toggle Hotel Status
* **Method:** `PATCH`
* **Path:** `/api/superadmin/hotels/:id/status`
* **Request Body:** `{"status": "inactive"}` (or empty body to toggle).

### 2.6 Delete Hotel Property
* **Method:** `DELETE`
* **Path:** `/api/superadmin/hotels/:id`
* **Security Rules:** Blocks deletion with `400 Bad Request` if any active reservations (`confirmed` or `checked_in`) exist.

### 2.7 Hotel Administrator Management
* **List Admins:** `GET /api/superadmin/admins` (Query: `page`, `limit`, `search`, `status`, `hotelId`)
* **Create Admin:** `POST /api/superadmin/admins`
```json
{
  "name": "Priya Nair",
  "email": "priya@seasideparadise.com",
  "phone": "+91 98333 44444",
  "password": "SecurePassword123!",
  "hotelId": "hotel-uuid",
  "status": "active"
}
```
* **Update Admin:** `PUT /api/superadmin/admins/:id`
* **Toggle Admin Status:** `PATCH /api/superadmin/admins/:id/status`
* **Delete Admin:** `DELETE /api/superadmin/admins/:id`
* **Get Hotel's Assigned Admins:** `GET /api/superadmin/hotels/:id/admins`

---

## 3. Hotel Admin Inventory Endpoints (`/api/admin` & `/api/v1/hotel`)

*Required Role:* `hotel_admin` (Strictly scoped by property `hotel_id` from JWT context).

### 3.1 Room Categories (`/api/admin/room-categories`)

#### List Categories
* **Method:** `GET`
* **Path:** `/api/admin/room-categories` (Alias: `/api/v1/hotel/categories`)
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "cat-uuid",
      "hotelId": "hotel-uuid",
      "name": "Deluxe Room",
      "description": "Spacious deluxe room with sea view",
      "baseRate": 5500.00,
      "defaultRate": 5500.00,
      "maxOccupancy": 2,
      "bedType": "King",
      "size": 38,
      "roomSize": 38,
      "amenities": ["High-Speed WiFi", "Air Conditioning", "Smart TV", "Mini Bar"],
      "image": "https://images.unsplash.com/photo-deluxe",
      "totalRooms": 25
    }
  ]
}
```

#### Create Room Category
* **Method:** `POST`
* **Path:** `/api/admin/room-categories` (Alias: `/api/v1/hotel/categories`)
* **Request Body:**
```json
{
  "name": "Junior Suite",
  "description": "Executive suite with city skyline view",
  "defaultRate": 8500.00,
  "maxOccupancy": 3,
  "bedType": "King",
  "roomSize": 55,
  "amenities": ["WiFi", "Jacuzzi", "Smart TV", "Mini Bar", "Safe"],
  "images": ["https://images.unsplash.com/photo-suite"]
}
```
* **Validation:** Name must be unique within the hotel property.

#### Update Category
* **Method:** `PUT`
* **Path:** `/api/admin/room-categories/:id`

#### Delete Category
* **Method:** `DELETE`
* **Path:** `/api/admin/room-categories/:id`
* **Constraint:** Rejects with **`409 Conflict`** if existing rooms reference this category.

---

### 3.2 Physical Rooms (`/api/admin/rooms`)

#### List Rooms
* **Method:** `GET`
* **Path:** `/api/admin/rooms` (Alias: `/api/v1/hotel/rooms`)
* **Query Parameters:**
  * `status`: `available`, `occupied`, `reserved`, `dirty`, `cleaning`, `out_of_service`, `ALL`
  * `floor`: integer (e.g. `2`)
  * `categoryId`: UUID
  * `search`: text string (room number, notes)
  * `page`, `limit`
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "room-uuid",
      "hotelId": "hotel-uuid",
      "roomNumber": "101",
      "floor": 1,
      "categoryId": "cat-uuid",
      "categoryName": "Deluxe Room",
      "rate": 5500.00,
      "status": "available",
      "notes": "Corner room facing the beach",
      "currentGuest": null,
      "category": { ... }
    }
  ],
  "meta": { "total": 1, "page": 1, "limit": 50 }
}
```

#### Add Room
* **Method:** `POST`
* **Path:** `/api/admin/rooms` (Alias: `/api/v1/hotel/rooms`)
* **Request Body:**
```json
{
  "roomNumber": "105",
  "floor": 1,
  "categoryId": "cat-uuid",
  "rate": 5500.00,
  "status": "available",
  "notes": "Adjacent to pool"
}
```
* **Validation:** `roomNumber` must be unique within the property. `categoryId` must exist in this property.

#### Update Room
* **Method:** `PUT`
* **Path:** `/api/admin/rooms/:id`

#### Update Room Status (Housekeeping & State Machine)
* **Method:** `PATCH`
* **Path:** `/api/admin/rooms/:id/status`
* **Request Body:**
```json
{
  "status": "cleaning",
  "notes": "Deep clean requested after checkout",
  "assignedTo": "Meera Devi"
}
```
* **Operational States:** `available`, `occupied`, `reserved`, `dirty`, `cleaning`, `out_of_service`
* **Housekeeping Side-Effect:** Automatically inserts an audit record in `housekeeping_logs`.

#### Delete Room
* **Method:** `DELETE`
* **Path:** `/api/admin/rooms/:id`
* **Constraint:** Rejects with **`409 Conflict`** if the room has active or upcoming reservations (`confirmed` or `checked_in`).

### 3.3 Housekeeping Operations Board (`/api/admin/housekeeping` & `/api/v1/hotel/housekeeping`)

#### List Housekeeping Tasks
* **Method:** `GET`
* **Path:** `/api/admin/housekeeping` (Alias: `/api/v1/hotel/housekeeping`)
* **Query Parameters:** `status` (`pending`, `in_progress`, `clean`, `all`), `floor` (int), `search` (room number)
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "hk-uuid",
      "roomId": "room-uuid",
      "roomNumber": "103",
      "floor": 1,
      "categoryName": "Standard Room",
      "roomStatus": "dirty",
      "hkStatus": "pending",
      "taskStatus": "pending",
      "assignedTo": "Meera Devi",
      "lastCleaned": "2024-11-12T09:30:00Z",
      "notes": "Full clean required",
      "priority": "high",
      "updatedAt": "2024-11-20T10:00:00Z"
    }
  ]
}
```

#### Update Housekeeping Status & Staff Assignment
* **Method:** `PATCH`
* **Path:** `/api/admin/housekeeping/:roomId` (Alias: `/api/v1/hotel/housekeeping/:roomId`)
* **Request Body:**
```json
{
  "taskStatus": "completed",
  "assignedTo": "Meera Devi",
  "notes": "Room cleaned and restocked"
}
```
* **Operational Transitions:**
  * `completed` or `clean`: room status transitions `dirty`/`cleaning` -> `available`
  * `in_progress`: room status transitions -> `cleaning`
  * `pending`: room status transitions -> `dirty`
* **Side-Effects:** Automatically records operational log in `housekeeping_logs` and audit entry in `activity_logs`.

---

## 4. Public Guest Booking Engine (`/api/public` & `/api/v1/public`)

*Zero-barrier public access (no login or token required).*

### 4.1 Public Hotel Profile
* **Method:** `GET`
* **Path:** `/api/public/hotels/:hotelId` (Alias: `/api/v1/public/hotel`, `/api/v1/public/hotels/:hotelId`)
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "d0e1b2c3-4567-89ab-cdef-0123456789ab",
    "name": "Grand Horizon Hotel",
    "description": "Mumbai's iconic 5-star beachfront luxury",
    "address": "Marine Drive, Nariman Point",
    "city": "Mumbai",
    "state": "Maharashtra",
    "country": "India",
    "phone": "+91 22 6610 8888",
    "email": "contact@grandhorizon.com",
    "image": "https://images.unsplash.com/photo-1566073771259-6a8506099945",
    "starRating": 5,
    "rating": 5.0,
    "checkInTime": "14:00",
    "checkOutTime": "12:00",
    "amenities": ["High-Speed Free WiFi", "Infinity Swimming Pool", "24/7 Room Service"]
  }
}
```

### 4.2 Room Categories Catalog
* **Method:** `GET`
* **Path:** `/api/public/hotels/:hotelId/categories` (Aliases: `/api/public/hotels/:hotelId/rooms`, `/api/v1/public/categories`)
* **Success Response (200 OK):** Returns array of room categories with available room counts.

### 4.3 Public Room & Category Detail
* **Method:** `GET`
* **Path:** `/api/v1/public/rooms/:id` (Aliases: `/api/public/rooms/:id`, `/api/v1/public/categories/:id`)
* **Description:** Retrieves room category overview, pricing, photos, and available room info for public guest browsing (`RoomDetailPage.jsx`).
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": "cat-uuid",
    "name": "Deluxe Sea View Room",
    "description": "Panoramic sea view with king bed",
    "defaultRate": 5000.00,
    "rate": 5000.00,
    "maxOccupancy": 2,
    "bedType": "King",
    "roomSize": 35,
    "amenities": ["WiFi", "Air Conditioning", "Smart TV"],
    "images": ["https://images.unsplash.com/photo-deluxe"],
    "roomNumber": "101"
  }
}
```

### 4.4 Check Room Availability (Date-Overlap Engine)
* **Method:** `GET`
* **Path:** `/api/public/hotels/:hotelId/availability` (Alias: `/api/v1/public/rooms`)
* **Query Parameters:**
  * `checkIn` (string, required): `YYYY-MM-DD` (e.g. `2024-12-01`)
  * `checkOut` (string, required): `YYYY-MM-DD` (e.g. `2024-12-04`)
  * `guests` (int, default: 1)
  * `categoryId` (UUID, optional)
* **Date Overlap Algorithm:**
  $$\text{Overlap} \iff \text{existing.checkIn} < \text{req.checkOut} \land \text{existing.checkOut} > \text{req.checkIn}$$
  *Rooms booked in conflicting intervals are filtered out. Same-day turnaround (checkout at 11:00 AM, new check-in at 14:00 PM) is fully supported.*
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "room-102-uuid",
      "roomNumber": "102",
      "floor": 1,
      "rate": 5000.00,
      "status": "available",
      "category": {
        "id": "cat-uuid",
        "name": "Deluxe Sea View Room",
        "baseRate": 5000.00,
        "maxOccupancy": 2,
        "bedType": "King"
      }
    }
  ],
  "meta": {
    "total": 1,
    "checkIn": "2024-12-01",
    "checkOut": "2024-12-04",
    "nights": 3,
    "guests": 2
  }
}
```

### 4.4 Create Guest Reservation (12% GST Billing Automation)
* **Method:** `POST`
* **Path:** `/api/public/bookings` (Alias: `/api/v1/public/bookings`)
* **Request Body:**
```json
{
  "hotelId": "d0e1b2c3-4567-89ab-cdef-0123456789ab",
  "roomId": "room-uuid",
  "categoryId": "cat-uuid",
  "guestName": "Priya Sharma",
  "guestEmail": "priya.sharma@example.com",
  "guestPhone": "+91 98700 12345",
  "idType": "Aadhaar",
  "idNumber": "AADHAAR-1234-5678-9012",
  "checkIn": "2024-12-10",
  "checkOut": "2024-12-13",
  "numGuests": 2,
  "specialRequests": "Upper floor sea view"
}
```
* **Pricing & Tax Formula:**
  $$\text{Subtotal} = \text{RatePerNight} \times \text{Nights}$$
  $$\text{Tax (12\% GST)} = \text{Round}(\text{Subtotal} \times 0.12)$$
  $$\text{Total Amount} = \text{Subtotal} + \text{Tax}$$
* **Success Response (201 Created):**
```json
{
  "success": true,
  "message": "Booking successfully confirmed",
  "data": {
    "id": "res-uuid",
    "bookingReference": "BK-2024-77192",
    "status": "confirmed",
    "hotelName": "Grand Horizon Hotel",
    "roomNumber": "101",
    "roomCategory": "Deluxe Sea View Room",
    "categoryName": "Deluxe Sea View Room",
    "guestName": "Priya Sharma",
    "guestEmail": "priya.sharma@example.com",
    "guestPhone": "+91 98700 12345",
    "checkIn": "2024-12-10",
    "checkOut": "2024-12-13",
    "nights": 3,
    "numGuests": 2,
    "ratePerNight": 5000.00,
    "roomRate": 5000.00,
    "subtotal": 15000.00,
    "taxRate": 0.12,
    "taxAmount": 1800.00,
    "totalAmount": 16800.00,
    "grandTotal": 16800.00,
    "currency": "INR",
    "specialRequests": "Upper floor sea view"
  }
}
```
* **Side-Effects:**
  1. Inserts confirmed record in `reservations`.
  2. Automatically creates customer invoice folio in `bills` with itemized charges in `bill_items`.
  3. Updates or creates profile in `guests`.
  4. Records event in `activity_logs`.
* **Error 409 Conflict:** If room was booked concurrently by another guest for conflicting dates.

### 4.5 Booking Confirmation Lookup
* **Method:** `GET`
* **Path:** `/api/public/bookings/:bookingReference`
* **Success Response (200 OK):** Returns full confirmed reservation matching the schema above.
* **Error 404:** Non-existent booking reference.


---

## 5. Front Desk Reservations & Room Operations (`/api/admin/reservations`)

Requires `Authorization: Bearer <HotelAdmin_JWT>`. Also accessible under enterprise alias `/api/v1/hotel/reservations`.

### 5.1 List Reservations
* **Method:** `GET`
* **Path:** `/api/admin/reservations`
* **Query Parameters:**
  * `status` (string, optional): Filter by `confirmed`, `checked_in`, `checked_out`, `cancelled`.
  * `search` (string, optional): Search by guest name, email, phone, or booking reference.
  * `roomId` (UUID, optional): Filter bookings assigned to a specific room.
  * `checkIn` (YYYY-MM-DD, optional): Filter by check-in date.
  * `checkOut` (YYYY-MM-DD, optional): Filter by check-out date.
  * `page` (int, default: 1): Page number.
  * `limit` (int, default: 20): Results per page.
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "res-uuid-1",
      "bookingReference": "BK-2024-77192",
      "hotelId": "hotel-uuid",
      "roomId": "room-uuid",
      "roomNumber": "101",
      "categoryId": "cat-uuid",
      "categoryName": "Deluxe Sea View Room",
      "guestName": "Priya Sharma",
      "guestEmail": "priya.sharma@example.com",
      "guestPhone": "+91 98700 12345",
      "checkIn": "2024-12-10",
      "checkOut": "2024-12-13",
      "nights": 3,
      "numGuests": 2,
      "totalAmount": 16800.00,
      "status": "confirmed",
      "specialRequests": "Upper floor"
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "limit": 20
  }
}
```

### 5.2 Create Front Desk / Walk-In Reservation
* **Method:** `POST`
* **Path:** `/api/admin/reservations`
* **Request Body:**
```json
{
  "roomId": "room-uuid",
  "guestName": "Rahul Verma",
  "guestEmail": "rahul.verma@example.com",
  "guestPhone": "+91 98111 22334",
  "idType": "Passport",
  "idNumber": "Z1234567",
  "checkIn": "2024-12-15",
  "checkOut": "2024-12-18",
  "numGuests": 2,
  "specialRequests": "Late check-in requested"
}
```
* **Success Response (201 Created):** Returns the generated reservation with auto-calculated 12% GST and initial customer folio in `bills`.

### 5.3 Get Reservation by ID
* **Method:** `GET`
* **Path:** `/api/admin/reservations/:id`
* **Success Response (200 OK):** Returns single reservation object including preloaded Room, Category, Guest details, and associated Bill folio.

### 5.4 Update Reservation
* **Method:** `PUT`
* **Path:** `/api/admin/reservations/:id`
* **Request Body:**
```json
{
  "roomId": "new-room-uuid",
  "numGuests": 3,
  "specialRequests": "Extra cot needed"
}
```
* **Success Response (200 OK):** Updated reservation details.

### 5.5 Check-In Guest
* **Method:** `PATCH`
* **Path:** `/api/admin/reservations/:id/check-in`
* **Description:** Performs front-desk guest check-in. Transitions reservation status to `checked_in` and physical room status to `occupied`.
* **Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Guest checked in successfully",
  "data": {
    "id": "res-uuid-1",
    "status": "checked_in",
    "roomStatus": "occupied"
  }
}
```

### 5.6 Check-Out Guest
* **Method:** `PATCH`
* **Path:** `/api/admin/reservations/:id/check-out`
* **Description:** Performs guest check-out. Transitions reservation status to `checked_out`, physical room status to `dirty`, and automatically inserts an operational log into `housekeeping_logs`.
* **Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Guest checked out successfully",
  "data": {
    "id": "res-uuid-1",
    "status": "checked_out",
    "roomStatus": "dirty"
  }
}
```

### 5.7 Cancel Reservation
* **Method:** `PATCH`
* **Path:** `/api/admin/reservations/:id/cancel`
* **Request Body (optional):**
```json
{
  "reason": "Guest emergency cancellation"
}
```
* **Description:** Transitions reservation status to `cancelled` and marks physical room `available`.
* **Success Response (200 OK):** Confirmation with updated status.

### 5.8 Today's Arrivals and Departures
* **Arrivals:** `GET /api/admin/reservations/today/check-ins` — List all reservations scheduled for arrival today.
* **Departures:** `GET /api/admin/reservations/today/check-outs` — List all checked-in reservations departing today.

---

## 6. Billing, Invoicing & Payments (`/api/admin/bills`, `/api/admin/payments`)

Requires `Authorization: Bearer <HotelAdmin_JWT>`. Also accessible under enterprise alias `/api/v1/hotel/bills` and `/api/v1/hotel/payments`.

### 6.1 List Bills / Folios
* **Method:** `GET`
* **Path:** `/api/admin/bills`
* **Query Parameters:** `status` (`unpaid`, `partial`, `paid`), `search`, `reservationId`, `page`, `limit`.
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "bill-uuid-1",
      "hotelId": "hotel-uuid",
      "reservationId": "res-uuid-1",
      "invoiceNumber": "INV-2024-00101",
      "subtotal": 15000.00,
      "taxRate": 0.12,
      "taxAmount": 1800.00,
      "discount": 0.00,
      "totalAmount": 16800.00,
      "amountPaid": 5000.00,
      "balance": 11800.00,
      "status": "partial",
      "currency": "INR",
      "guestName": "Priya Sharma",
      "roomNumber": "101",
      "createdAt": "2024-12-10T14:30:00Z"
    }
  ],
  "meta": { "total": 1, "page": 1, "limit": 20 }
}
```

### 6.2 Get Bill by ID
* **Method:** `GET`
* **Path:** `/api/admin/bills/:id`
* **Success Response (200 OK):** Returns detailed bill object with preloaded line items (`items`), payments (`payments`), reservation, and guest data.

### 6.3 Post Line Item / Facility Charge to Folio
* **Method:** `POST`
* **Path:** `/api/admin/bills/:id/items`
* **Description:** Appends additional stay charges (e.g., room service, laundry, spa, minibar). Automatically calculates 12% GST on the new item, recalculates subtotal, totalAmount, and balance.
* **Request Body:**
```json
{
  "itemType": "restaurant",
  "description": "Room Service Dinner - Day 1",
  "quantity": 2,
  "unitPrice": 600.00
}
```
* **Success Response (201 Created):** Returns updated bill folio with recalculated financial totals.

### 6.4 Apply Discount
* **Method:** `PATCH`
* **Path:** `/api/admin/bills/:id/discount`
* **Request Body:**
```json
{
  "discount": 1000.00
}
```
* **Success Response (200 OK):** Returns updated bill with reduced `totalAmount` and `balance`.

### 6.5 Record Payment
* **Method:** `POST`
* **Path:** `/api/admin/payments` (Aliases: `/api/v1/hotel/payments`, `/api/admin/bills/:id/payments`, `/api/v1/hotel/bills/:id/payments`)
* **Request Body:**
```json
{
  "billId": "bill-uuid-1",
  "amount": 11800.00,
  "paymentMethod": "upi",
  "transactionId": "UPI-IND-9876543210",
  "notes": "Settled remaining balance at checkout"
}
```
* **Success Response (201 Created / 200 OK):**
```json
{
  "success": true,
  "message": "Payment recorded successfully",
  "data": {
    "id": "payment-uuid",
    "billId": "bill-uuid-1",
    "amount": 11800.00,
    "paymentMethod": "upi",
    "transactionId": "UPI-IND-9876543210",
    "status": "completed",
    "billStatus": "paid",
    "remainingBalance": 0.00
  }
}
```
* **Side-Effects:**
  1. Creates ledger entry in `payments`.
  2. Increments `amountPaid` on `bills`.
  3. Reduces `balance` on `bills`.
  4. Automatically transitions bill `status` to `paid` if balance <= 0, or `partial` if balance > 0.
  5. Records audit entry in `activity_logs`.

### 6.6 Print Bill Folio with Letterhead
* **Method:** `GET`
* **Path:** `/api/admin/bills/:id/print` (Alias: `/api/v1/hotel/bills/:id/print`)
* **Success Response (200 OK):** Returns complete itemized invoice merged with hotel company letterhead metadata (`name`, `address`, `phone`, `email`, `checkInTime`, `checkOutTime`) for `PrintBill.jsx`.

### 6.7 Facility Charges (Running Folio)
* **List Facility Charges:** `GET /api/admin/facility-charges` (Alias: `/api/v1/hotel/facility-charges`, Query: `reservationId`, `category`, `search`)
* **Create Facility Charge:** `POST /api/admin/facility-charges` (Alias: `/api/v1/hotel/facility-charges`)
```json
{
  "reservationId": "res-uuid",
  "category": "restaurant",
  "description": "Room Service Dinner - Day 1",
  "amount": 1200.00
}
```
*Auto-appends item to guest's running bill folio and recalculates 12% GST.*
* **Delete Facility Charge:** `DELETE /api/admin/facility-charges/:id` (Alias: `/api/v1/hotel/facility-charges/:id`)

---

## 7. Hotel Admin Reporting & Live Analytics (`/api/admin/dashboard`, `/api/admin/reports/*`)

Requires `Authorization: Bearer <HotelAdmin_JWT>`. Also accessible under enterprise alias `/api/v1/hotel/reports/*`.

### 7.1 Hotel Admin Dashboard Snapshot
* **Method:** `GET`
* **Path:** `/api/admin/dashboard`
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "hotelId": "hotel-uuid",
    "hotelName": "Grand Horizon Hotel",
    "totalRooms": 50,
    "availableRooms": 32,
    "occupiedRooms": 14,
    "dirtyRooms": 4,
    "outOfServiceRooms": 0,
    "occupancyRate": 28.0,
    "todayArrivals": 6,
    "todayDepartures": 4,
    "monthlyRevenue": 485000.00,
    "totalActiveBookings": 14,
    "recentReservations": [ ... ]
  }
}
```

### 7.2 Occupancy Report
* **Method:** `GET`
* **Path:** `/api/admin/reports/occupancy`
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "totalRooms": 50,
    "occupiedRooms": 14,
    "availableRooms": 32,
    "dirtyRooms": 4,
    "occupancyRate": 28.0,
    "byCategory": [
      {
        "categoryName": "Deluxe Sea View Room",
        "total": 20,
        "occupied": 8,
        "occupancyRate": 40.0
      }
    ]
  }
}
```

### 7.3 Revenue Report
* **Method:** `GET`
* **Path:** `/api/admin/reports/revenue`
* **Query Parameters:** `startDate` (YYYY-MM-DD), `endDate` (YYYY-MM-DD).
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "currency": "INR",
    "totalRevenue": 485000.00,
    "totalTaxCollected": 58200.00,
    "totalDiscountGiven": 12000.00,
    "totalOutstandingBalance": 34000.00,
    "byPaymentMethod": {
      "upi": 285000.00,
      "credit_card": 140000.00,
      "cash": 60000.00
    }
  }
}
```

### 7.4 Housekeeping Log & Operational Report
* **Method:** `GET`
* **Path:** `/api/admin/reports/housekeeping`
* **Success Response (200 OK):** Returns log history of room turnovers, cleaning assignments, and maintenance alerts.

### 7.5 Booking Distribution Report
* **Method:** `GET`
* **Path:** `/api/admin/reports/bookings` (Alias: `/api/v1/hotel/reports/bookings`, `/api/v1/hotel/reports/reservations`, `/api/admin/reports/reservations`)
* **Success Response (200 OK):** Returns breakdown of bookings by status (`confirmed`, `checked_in`, `checked_out`, `cancelled`) and time distribution.

### 7.6 Room Status Breakdown Report
* **Method:** `GET`
* **Path:** `/api/admin/reports/room-status` (Alias: `/api/v1/hotel/reports/room-status`)
* **Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "available": 45,
    "occupied": 55,
    "reserved": 15,
    "dirty": 3,
    "cleaning": 2,
    "outOfService": 0,
    "total": 120
  }
}
```
