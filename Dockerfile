# syntax=docker/dockerfile:1

# -- Build stage --
FROM golang:1.24.5-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /main cmd/main/main.go

# -- Final stage -- 
FROM alpine:3

EXPOSE 8080

COPY --from=builder main .

ENTRYPOINT [ "./main" ]