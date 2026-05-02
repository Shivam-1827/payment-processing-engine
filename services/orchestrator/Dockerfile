FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY services/orchestrator/ services/orchestrator/
RUN CGO_ENABLED=0 GOOS=linux go build -o orchestrator ./services/orchestrator/cmd/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/orchestrator .
CMD ["./orchestrator"]
