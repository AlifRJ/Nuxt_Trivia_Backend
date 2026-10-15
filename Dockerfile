# Build Stage
FROM golang:1.27.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main .

# Final Stage (Minimal Image)
FROM alpine:latest

# Install CA certificates for outgoing TLS/HTTPS connections
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user for security
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Make sure bin is owned by appuser
RUN chown appuser:appuser main

# Switch to non-root user
USER appuser

EXPOSE 8080
CMD ["./main"]