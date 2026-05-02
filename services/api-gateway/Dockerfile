# Build Stage
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY services/api-gateway/ services/api-gateway/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway ./services/api-gateway/cmd/main.go

# Runtime Stage (Minimal Alpine Image)
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/api-gateway .
EXPOSE 8080
CMD ["./api-gateway"]