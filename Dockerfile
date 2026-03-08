# Stage 1: Build the binary
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git and ca-certificates (needed for some Go modules)
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

# Copy go.mod and go.sum first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source code
COPY . .

# Build the binary - compile main.go directly
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o /app/supervisor ./main.go

# Stage 2: Minimal runner
FROM alpine:latest

# Certificates and timezone data are required
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/supervisor .

CMD ["./supervisor"]
