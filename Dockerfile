# syntax=docker/dockerfile:1

# -- Build stage --
FROM golang:1.24.3-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o /main cmd/main/main.go

# -- Final stage -- 
FROM alpine:3

COPY --from=builder main /bin/main
ENTRYPOINT [ "/bin/main" ]