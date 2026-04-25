FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/order-service ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/migrate ./cmd/migrate

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /bin/order-service /usr/local/bin/order-service
COPY --from=builder /bin/migrate /usr/local/bin/migrate
COPY --from=builder /src/database /app/database

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/order-service"]
