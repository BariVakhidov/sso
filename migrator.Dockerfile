# syntax=docker/dockerfile:1

# Build the application from source
FROM golang:1.22.2 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod verify

COPY .  ./

# Build migrator
RUN CGO_ENABLED=0 GOOS=linux go build -o /migrator ./cmd/migrator

FROM alpine:latest

WORKDIR /root/

COPY --from=build-stage /migrator /migrator
COPY --from=build-stage /app/config /config
COPY --from=build-stage /app/migrations /migrations

ENTRYPOINT [ "/migrator","--db","postgres", "--migrations-path","../migrations/postgres/", "--storage-path","db:5432" ]