# Build stage
FROM golang:1.24-alpine AS builder

# Install git and ca-certificates for HTTPS
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o gitlab-smith ./cmd/gitlab-smith

# Final stage
FROM scratch

# Copy ca-certificates and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary from builder
COPY --from=builder /app/gitlab-smith /gitlab-smith

# Set the entrypoint
ENTRYPOINT ["/gitlab-smith"]

# Default command
CMD ["--help"]