FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY . .

# Build with security flags enabled
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -trimpath \
    -o socks5-proxy

FROM alpine:latest
RUN adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /build/socks5-proxy .

# Set proper permissions
RUN chown appuser:appuser /app/socks5-proxy && \
    chmod 500 /app/socks5-proxy && \
    mkdir -p /home/appuser/.go-socks5-chain && \
    chown appuser:appuser /home/appuser/.go-socks5-chain

USER appuser

EXPOSE 1080
ENTRYPOINT ["/app/socks5-proxy"]
