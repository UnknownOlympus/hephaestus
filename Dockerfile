# syntax=docker/dockerfile:1

# -- Migrator stage --
FROM golang:1.24.3-alpine AS migrator

WORKDIR /build
COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /migrator ./cmd/migrator/main.go

# -- Build stage --
FROM golang:1.24.3-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /main cmd/main/main.go

# -- Final stage -- 
FROM alpine:3

COPY --from=builder main .
COPY --from=migrator migrator .
COPY ./migrations ./migrations

ENTRYPOINT [ "./main" ]