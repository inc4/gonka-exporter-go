FROM golang:1.22-alpine AS builder

WORKDIR /build

# Download dependencies first (cache layer)
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o gonka-exporter ./cmd/exporter/

# -------- runtime --------
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/gonka-exporter .

# Default data directory
VOLUME ["/data"]

EXPOSE 9404

ENTRYPOINT ["/app/gonka-exporter"]
