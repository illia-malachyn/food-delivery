FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY restaurant/go.mod restaurant/go.sum ./restaurant/

WORKDIR /src/restaurant
RUN go mod download

WORKDIR /src
COPY . .

WORKDIR /src/restaurant
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/restaurant-service ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/migrate ./cmd/migrate

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /bin/restaurant-service /usr/local/bin/restaurant-service
COPY --from=builder /bin/migrate /usr/local/bin/migrate
COPY --from=builder /src/restaurant/database /app/database
COPY --from=builder /src/restaurant/keys /app/keys

ENTRYPOINT ["/usr/local/bin/restaurant-service"]
