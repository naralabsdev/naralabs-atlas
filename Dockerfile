# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/atlas .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/atlas /app/atlas
COPY db/migrations ./db/migrations

ENV MIGRATIONS_DIR=/app/db/migrations

USER nobody

ENTRYPOINT ["/app/atlas", "worker"]
