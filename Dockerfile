# Stage 1: Build the binary
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary from the root main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/supervisor main.go

# Stage 2: Minimal runner
FROM alpine:latest

# Certificates and timezone data are required
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/supervisor .

CMD ["./supervisor"]
