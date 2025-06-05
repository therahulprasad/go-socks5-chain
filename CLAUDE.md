# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-socks5-chain is a SOCKS5 proxy server that forwards all client connections through an upstream SOCKS5 proxy. It features secure credential storage using AES-GCM encryption and supports proxy chaining scenarios.

## Common Development Commands

### Build Commands
```bash
# Build for current platform
go build -o go-socks5-chain

# Build with optimization flags (production)
go build -ldflags="-w -s" -trimpath -o go-socks5-chain

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o go-socks5-chain
GOOS=darwin GOARCH=arm64 go build -o go-socks5-chain-mac
GOOS=windows GOARCH=amd64 go build -o go-socks5-chain.exe
```

### Run and Test
```bash
# Run with basic configuration
./go-socks5-chain --upstream-host proxy.example.com --upstream-port 1080 --username user --password pass --encpass myencryptionpass

# Interactive configuration mode
./go-socks5-chain --configure

# Run with environment variables
export UPSTREAM_USERNAME=myuser
export UPSTREAM_PASSWORD=mypass
export SOCKS5CHAIN_PASSWORD=myencryptionpass
./go-socks5-chain --upstream-host proxy.example.com --upstream-port 1080
```

### Docker Commands
```bash
# Build Docker image (Linux)
docker build -t go-socks5-chain .

# Build for specific platforms
docker build -f Dockerfile.mac-arm64 -t go-socks5-chain-mac .
docker build -f Dockerfile.windows -t go-socks5-chain-windows .

# Run Docker container
docker run -p 1080:1080 -v socks5-config:/home/appuser/.go-socks5-chain go-socks5-chain \
  --upstream-host proxy.example.com --upstream-port 1080
```

## Code Architecture

### Key Components

1. **main.go**: Entry point handling CLI flags, configuration, and graceful shutdown
   - Version management (currently v0.1)
   - Interactive configuration mode
   - Signal handling for graceful shutdown
   - Logging setup (file or console)

2. **proxy/proxy.go**: Core SOCKS5 server implementation
   - SOCKS5 protocol handshake
   - Connection management between client and upstream proxy
   - Bidirectional traffic forwarding
   - IPv4, IPv6, and domain name support

3. **config/config.go**: Secure credential management
   - AES-GCM encryption for credentials
   - Configuration stored in `~/.go-socks5-chain/`
   - Password-based key derivation using SHA256

### Important Implementation Details

- **Credential Storage**: Credentials are encrypted using AES-GCM and stored in `~/.go-socks5-chain/upstream_creds.enc`
- **Password Handling**: The encryption password can be provided via `--encpass` flag or `SOCKS5CHAIN_PASSWORD` environment variable
- **Graceful Shutdown**: Server properly closes all connections with a 5-second timeout on SIGINT/SIGTERM
- **Error Handling**: All errors are propagated and logged appropriately
- **Platform Support**: Special handling for Windows, macOS, and Linux configuration directories

### Security Considerations

- Credentials are never stored in plaintext
- Multi-stage Docker builds with non-root users
- Minimal file permissions (500) on binaries
- Secure password input using golang.org/x/term