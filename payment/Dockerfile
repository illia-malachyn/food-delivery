FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY payment/go.mod payment/go.sum ./payment/

WORKDIR /src/payment
RUN go mod download

WORKDIR /src
COPY . .

WORKDIR /src/payment
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/payment-service ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/migrate ./cmd/migrate

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /bin/payment-service /usr/local/bin/payment-service
COPY --from=builder /bin/migrate /usr/local/bin/migrate
COPY --from=builder /src/payment/database /app/database

ENTRYPOINT ["/usr/local/bin/payment-service"]
