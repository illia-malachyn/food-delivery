FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/order-service ./cmd/app

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /bin/order-service /usr/local/bin/order-service

EXPOSE 9876

ENTRYPOINT ["/usr/local/bin/order-service"]
