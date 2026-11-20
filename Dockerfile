# ============================================================
# Stage 1: Build the Go application
# ============================================================
FROM golang:1.26-alpine AS builder

# Install certificates and git
RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /app

# Leverage caching for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source
COPY . .

# Build static Linux binary (disabling CGO for a pure, lightweight binary)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/server ./cmd/server

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.20

WORKDIR /app

# Install runtime certificates and Indian timezone
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Kolkata

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Expose HTTP API port
EXPOSE 8080

# Run the backend
CMD ["/app/server"]
