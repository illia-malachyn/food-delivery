FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY delivery/go.mod delivery/go.sum ./delivery/

WORKDIR /src/delivery
RUN go mod download

WORKDIR /src
COPY . .

WORKDIR /src/delivery
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/delivery-service ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/migrate ./cmd/migrate

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /bin/delivery-service /usr/local/bin/delivery-service
COPY --from=builder /bin/migrate /usr/local/bin/migrate
COPY --from=builder /src/delivery/database /app/database

ENTRYPOINT ["/usr/local/bin/delivery-service"]
