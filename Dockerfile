# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o smtp2discord ./cmd/smtp2discord

# Final stage – minimal image with no shell or OS packages
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/smtp2discord /smtp2discord
USER 65534:65534
ENTRYPOINT ["/smtp2discord"]
