# Multi-stage build for minimal image size

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the agent binary
# CGO is disabled for a fully static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o saviour-agent \
    ./cmd/agent

# Final stage - minimal runtime image
FROM scratch

# Copy CA certificates for HTTPS requests (future use)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /build/saviour-agent /saviour-agent

# Copy example config (can be overridden with volume mount)
COPY --from=builder /build/examples/agent.yaml /etc/saviour/agent.yaml

# Use non-root user (numeric UID for scratch image)
USER 65534:65534

# Set default config path
ENV CONFIG_PATH=/etc/saviour/agent.yaml

# Run the agent
ENTRYPOINT ["/saviour-agent"]
CMD ["-config", "/etc/saviour/agent.yaml"]
