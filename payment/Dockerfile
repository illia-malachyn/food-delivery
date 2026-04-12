FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod .
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/payment-service ./cmd/app

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /bin/payment-service /usr/local/bin/payment-service

ENTRYPOINT ["/usr/local/bin/payment-service"]
