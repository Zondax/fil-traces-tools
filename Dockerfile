# Multi-stage build for fil-trace-check

# Build stage
FROM golang:1.24.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o fil-trace-check .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/fil-trace-check /app/

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Run the binary
ENTRYPOINT ["./fil-trace-check"]