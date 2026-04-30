FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY auth/go.mod auth/go.sum ./auth/

WORKDIR /src/auth
RUN go mod download

WORKDIR /src
COPY . .

WORKDIR /src/auth
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/auth-service ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/migrate ./cmd/migrate

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /bin/auth-service /usr/local/bin/auth-service
COPY --from=builder /bin/migrate /usr/local/bin/migrate
COPY --from=builder /src/auth/database /app/database
COPY --from=builder /src/auth/keys /app/keys

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/auth-service"]
