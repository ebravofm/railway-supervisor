# Stage 1: Build the binary
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy dependency files (if any)
COPY go.mod ./
# Copy source code
COPY main.go ./

# Build an optimized, static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o supervisor main.go

# Stage 2: Minimal runner
FROM alpine:latest

# Certificates and timezone data are required
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/supervisor .

CMD ["./supervisor"]
