# SOCKS5 Upstream Proxy Chain (Go Implementation)

A Go SOCKS5 proxy server that forwards all client connections through a configurable upstream SOCKS5 proxy. This project is designed for privacy, chaining proxies, or routing traffic through a remote SOCKS5 endpoint.

## Building and Running

### Requirements
- Go 1.19 or newer
- Linux, macOS, or Windows

### Building from Source
```sh
go build
```

### Configuration Options

There are two ways to configure the proxy:

1. **Interactive Configuration Mode**:
```sh
./socks5-proxy --configure --upstream-host proxy.example.com --upstream-port 1080
```
This will:
- Prompt for username, password, and encryption password
- Save the encrypted credentials
- Start the proxy server

2. **Command Line Arguments**:
```sh
./socks5-proxy --username myuser --password mypass --upstream-host proxy.example.com --upstream-port 1080
```

### Command Line Options
- `--configure`        Enable interactive configuration mode
- `--username`        Upstream SOCKS5 username (can also use env var `UPSTREAM_USERNAME`)
- `--password`        Upstream SOCKS5 password (can also use env var `UPSTREAM_PASSWORD`)
- `--encpass`         Password to encrypt/decrypt stored credentials
- `--upstream-host`   Upstream SOCKS5 proxy hostname (required on first run)
- `--upstream-port`   Upstream SOCKS5 proxy port (required on first run)
- `--local-host`      Local host to bind the proxy server (default: 127.0.0.1)
- `--local-port`      Local port to bind the proxy server (default: 1080)
- `--log-file`        Log file location (default: no file logging)
- `--console-log`     Enable logging to terminal (default: off)

### Stored Credentials
Once configured, credentials are stored securely in `~/.go-socks5-chain/`:
- The proxy settings are stored in `upstream_config`
- Encrypted credentials are stored in `upstream_creds.enc`

For subsequent runs, you only need to provide the encryption password:
```sh
./socks5-proxy --encpass mypass
```

If you don't provide the encryption password, you'll be prompted for it:
```sh
./socks5-proxy
```

### Environment Variables
You can also set credentials via environment variables:
```sh
export UPSTREAM_USERNAME=myuser
export UPSTREAM_PASSWORD=mypass
./socks5-proxy --upstream-host proxy.example.com --upstream-port 1080
```

## Docker Support

The project includes platform-specific Dockerfiles for Linux, Windows, and macOS (Apple Silicon).

### Building for Linux (amd64/arm64)
```sh
docker build -t socks5-proxy-linux -f Dockerfile .
```

### Building for Windows
```powershell
# Switch to Windows containers first
docker build -t socks5-proxy-windows -f Dockerfile.windows .
```

### Building for macOS (Apple Silicon)
```sh
docker build -t socks5-proxy-mac -f Dockerfile.mac-arm64 .
```

### Running Docker Containers

#### Linux
```sh
docker run --rm -it -p 1080:1080 \
  -v $HOME/.go-socks5-chain:/root/.go-socks5-chain \
  socks5-proxy-linux \
  --upstream-host proxy.example.com --upstream-port 1080 --console-log
```

#### Windows
```powershell
docker run --rm -it -p 1080:1080 `
  -v $env:USERPROFILE\AppData\Local\go-socks5-chain:C:\Users\ContainerUser\AppData\Local\go-socks5-chain `
  socks5-proxy-windows `
  --upstream-host proxy.example.com --upstream-port 1080 --console-log
```

#### macOS (Apple Silicon)
```sh
docker run --rm -it -p 1080:1080 \
  -v "$HOME/Library/Application Support/go-socks5-chain:/Users/appuser/Library/Application Support/go-socks5-chain" \
  socks5-proxy-mac \
  --upstream-host proxy.example.com --upstream-port 1080 --console-log
```

### Native Builds

You can also build and run the application natively for each platform:

#### Linux
```sh
GOOS=linux GOARCH=amd64 go build -o socks5-proxy
```

#### Windows
```sh
GOOS=windows GOARCH=amd64 go build -o socks5-proxy.exe
```

#### macOS (Apple Silicon)
```sh
GOOS=darwin GOARCH=arm64 go build -o socks5-proxy
```

## Security Note
- Credentials are encrypted using AES-GCM with a key derived from your encryption password
- The encryption password is never stored, you must provide it each time
- If you forget your encryption password, you'll need to reconfigure the proxy

## Features
- SOCKS5 protocol support
- Secure credential storage using AES encryption
- Interactive configuration mode
- Command-line and environment variable support
- Docker support
- Configurable logging

## License
MIT License
