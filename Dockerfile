# Step 1: Build the Go binary
FROM golang:1.24.2 AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build a Linux binary
RUN GOOS=linux GOARCH=amd64 go build -o authllama ./cmd/server

# Step 2: Minimal runtime image
FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/authllama .

# Run the server
CMD ["./authllama"]