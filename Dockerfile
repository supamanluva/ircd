# Multi-stage build for smaller final image
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ircd ./cmd/ircd

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 ircd && \
    adduser -D -u 1000 -G ircd ircd

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/ircd .

# Copy configuration
COPY --from=builder /build/config/config.yaml ./config/

# Create logs directory
RUN mkdir -p logs && chown -R ircd:ircd /app

# Switch to non-root user
USER ircd

# Expose IRC ports
EXPOSE 6667 6697 8080

# Run the server
CMD ["./ircd", "-config", "config/config.yaml"]
