# Stage 1: Build
FROM golang:1.26-alpine AS builder

# Install build dependencies (needed for CGO and future usearch-go)
RUN apk add --no-cache gcc musl-dev g++ make

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the core binary
# Note: CGO_ENABLED=1 because usearch-go will require it later
RUN CGO_ENABLED=1 GOOS=linux go build -o /vectora ./cmd/core/main.go

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates libc6-compat

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /vectora .

# Create data directory
RUN mkdir -p /root/.Vectora/data

# Expose HTTP port
EXPOSE 8080

# Environment variables
ENV VECTORA_MODE=service
ENV PORT=8080

# Entry point
ENTRYPOINT ["./vectora", "start"]
