package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"go-socks5-chain/config"
)

const (
	VERSION = 0x05
)

type Server struct {
	config   *config.Config
	listener net.Listener
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewServer(cfg *config.Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Server) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %v", err)
	}
	s.listener = listener

	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
			conn, err := listener.Accept()
			if err != nil {
				if err, ok := err.(*net.OpError); ok && err.Err.Error() == "use of closed network connection" {
					return nil
				}
				log.Printf("Failed to accept connection: %v", err)
				continue
			}

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) Stop() {
	// Signal shutdown
	s.cancel()

	// Close listener to stop accepting new connections
	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for existing connections with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All connections closed gracefully
	case <-time.After(5 * time.Second):
		// Timeout - some connections might still be active
		log.Println("Shutdown timeout - some connections may still be active")
	}
}

func (s *Server) handleConnection(client net.Conn) {
	defer client.Close()
	defer s.wg.Done()

	// SOCKS5 initial handshake
	if err := s.handleInitialHandshake(client); err != nil {
		log.Printf("Initial handshake failed: %v", err)
		return
	}

	// Handle SOCKS5 request
	target, err := s.handleRequest(client)
	if err != nil {
		log.Printf("Request handling failed: %v", err)
		return
	}

	// Connect to upstream proxy
	upstreamConn, err := s.connectToUpstream()
	if err != nil {
		log.Printf("Failed to connect to upstream: %v", err)
		return
	}
	defer upstreamConn.Close()

	// Forward the connection request to upstream
	if err := s.forwardRequest(upstreamConn, target); err != nil {
		log.Printf("Failed to forward request: %v", err)
		return
	}

	// Start bidirectional forwarding
	s.forwardTraffic(client, upstreamConn)
}

func (s *Server) handleInitialHandshake(conn net.Conn) error {
	// Read version and number of methods
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}

	if header[0] != VERSION {
		return fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	// Read authentication methods
	methods := make([]byte, header[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	// Respond with no authentication required
	_, err := conn.Write([]byte{VERSION, 0x00})
	return err
}

func (s *Server) handleRequest(conn net.Conn) (string, error) {
	// Read request header
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", err
	}

	if header[0] != VERSION {
		return "", fmt.Errorf("unsupported SOCKS version: %d", header[0])
	}

	// Read address type and address
	var addr string
	switch header[3] {
	case 0x01: // IPv4
		ipv4 := make([]byte, 4)
		if _, err := io.ReadFull(conn, ipv4); err != nil {
			return "", err
		}
		addr = net.IP(ipv4).String()
	case 0x03: // Domain name
		lenByte := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenByte); err != nil {
			return "", err
		}
		domain := make([]byte, lenByte[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		addr = string(domain)
	case 0x04: // IPv6
		ipv6 := make([]byte, 16)
		if _, err := io.ReadFull(conn, ipv6); err != nil {
			return "", err
		}
		addr = net.IP(ipv6).String()
	default:
		return "", fmt.Errorf("unsupported address type: %d", header[3])
	}

	// Read port
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBytes); err != nil {
		return "", err
	}
	port := int(portBytes[0])<<8 | int(portBytes[1])

	// Send success response
	response := []byte{VERSION, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}
	if _, err := conn.Write(response); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%d", addr, port), nil
}

func (s *Server) connectToUpstream() (net.Conn, error) {
	upstreamAddr := fmt.Sprintf("%s:%d", s.config.UpstreamHost, s.config.UpstreamPort)
	conn, err := net.Dial("tcp", upstreamAddr)
	if err != nil {
		return nil, err
	}

	// SOCKS5 handshake with upstream
	// Version + number of auth methods
	_, err = conn.Write([]byte{VERSION, 0x01, 0x02})
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Read auth method selection
	response := make([]byte, 2)
	if _, err := io.ReadFull(conn, response); err != nil {
		conn.Close()
		return nil, err
	}

	// Authenticate with upstream
	auth := []byte{0x01}                              // Username/Password auth version
	auth = append(auth, byte(len(s.config.Username))) // Username length
	auth = append(auth, []byte(s.config.Username)...) // Username
	auth = append(auth, byte(len(s.config.Password))) // Password length
	auth = append(auth, []byte(s.config.Password)...) // Password
	if _, err := conn.Write(auth); err != nil {
		conn.Close()
		return nil, err
	}

	// Read auth response
	authResponse := make([]byte, 2)
	if _, err := io.ReadFull(conn, authResponse); err != nil {
		conn.Close()
		return nil, err
	}

	if authResponse[1] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("upstream authentication failed")
	}

	return conn, nil
}

func (s *Server) forwardRequest(conn net.Conn, target string) error {
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		return err
	}

	// Build SOCKS5 connect request
	request := []byte{VERSION, 0x01, 0x00, 0x03}
	request = append(request, byte(len(host)))
	request = append(request, []byte(host)...)

	portNum := 0
	fmt.Sscanf(port, "%d", &portNum)
	request = append(request, byte(portNum>>8), byte(portNum&0xff))

	if _, err := conn.Write(request); err != nil {
		return err
	}

	// Read response
	response := make([]byte, 10)
	if _, err := io.ReadFull(conn, response); err != nil {
		return err
	}

	if response[1] != 0x00 {
		return fmt.Errorf("upstream connection failed: %d", response[1])
	}

	return nil
}

func (s *Server) forwardTraffic(client, upstream net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Upstream
	go func() {
		defer wg.Done()
		io.Copy(upstream, client)
		upstream.(*net.TCPConn).CloseWrite()
	}()

	// Upstream -> Client
	go func() {
		defer wg.Done()
		io.Copy(client, upstream)
		client.(*net.TCPConn).CloseWrite()
	}()

	wg.Wait()
}
