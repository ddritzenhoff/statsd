# syntax=docker/dockerfile:1

FROM golang:1.21 AS builder

WORKDIR /statsd

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /usr/local/bin/ ./...
RUN touch statsd.db

FROM builder AS tester
RUN go test -v ./...

FROM gcr.io/distroless/base-debian12

WORKDIR /

COPY --from=builder /usr/local/bin/statsd /usr/local/bin/statsd
COPY --from=builder --chown=nonroot:nonroot --chmod=0644 /statsd/statsd.db /data/statsd.db

EXPOSE 8080

USER nonroot:nonroot
