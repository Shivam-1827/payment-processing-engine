FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY services/webhook-service/ services/webhook-service/
RUN CGO_ENABLED=0 GOOS=linux go build -o webhook-service ./services/webhook-service/cmd/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/webhook-service .
CMD ["./webhook-service"]
