FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy only necessary files
COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY config/ config/
COPY proxy/ proxy/

# Build with security flags enabled
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -trimpath \
    -o go-socks5-chain

# Final stage
FROM alpine:latest
RUN adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /build/go-socks5-chain .

# Set proper permissions
RUN chown appuser:appuser /app/go-socks5-chain && \
    chmod 500 /app/go-socks5-chain && \
    mkdir -p /home/appuser/.go-socks5-chain && \
    chown -R appuser:appuser /home/appuser/.go-socks5-chain

USER appuser

EXPOSE 1080
ENTRYPOINT ["/app/go-socks5-chain"]
