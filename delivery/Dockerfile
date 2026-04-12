FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod .
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/delivery-service ./cmd/app

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /bin/delivery-service /usr/local/bin/delivery-service

ENTRYPOINT ["/usr/local/bin/delivery-service"]
