package proxy

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"go-socks5-chain/config"
)

// MockConn implements net.Conn for testing
type MockConn struct {
	readBuffer  *bytes.Buffer
	writeBuffer *bytes.Buffer
	closed      bool
	localAddr   net.Addr
	remoteAddr  net.Addr
	mu          sync.RWMutex
}

func NewMockConn() *MockConn {
	return &MockConn{
		readBuffer:  &bytes.Buffer{},
		writeBuffer: &bytes.Buffer{},
		localAddr:   &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345},
		remoteAddr:  &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54321},
	}
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return 0, io.EOF
	}
	return m.readBuffer.Read(b)
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.writeBuffer.Write(b)
}

func (m *MockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *MockConn) LocalAddr() net.Addr  { return m.localAddr }
func (m *MockConn) RemoteAddr() net.Addr { return m.remoteAddr }

func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

// Add data to be read by the connection
func (m *MockConn) AddReadData(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readBuffer.Write(data)
}

// Get data that was written to the connection
func (m *MockConn) GetWrittenData() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.writeBuffer.Bytes()
}

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		Username:     "testuser",
		Password:     "testpass",
		UpstreamHost: "proxy.example.com",
		UpstreamPort: 1080,
	}

	server := NewServer(cfg)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	if server.config != cfg {
		t.Error("Server config not set correctly")
	}
	if server.ctx == nil {
		t.Error("Server context not initialized")
	}
	if server.cancel == nil {
		t.Error("Server cancel function not initialized")
	}
}

func TestServerStop(t *testing.T) {
	cfg := &config.Config{
		Username:     "testuser",
		Password:     "testpass",
		UpstreamHost: "proxy.example.com",
		UpstreamPort: 1080,
	}

	server := NewServer(cfg)
	
	// Start a goroutine that waits for context cancellation
	done := make(chan bool)
	go func() {
		<-server.ctx.Done()
		done <- true
	}()

	// Stop the server
	server.Stop()

	// Verify context was cancelled
	select {
	case <-done:
		// Context was cancelled as expected
	case <-time.After(time.Second):
		t.Error("Server context was not cancelled")
	}
}

func TestHandleInitialHandshake(t *testing.T) {
	cfg := &config.Config{
		Username:     "testuser",
		Password:     "testpass",
		UpstreamHost: "proxy.example.com",
		UpstreamPort: 1080,
	}
	server := NewServer(cfg)

	tests := []struct {
		name      string
		input     []byte
		wantError bool
		expected  []byte
	}{
		{
			name:      "Valid handshake no auth",
			input:     []byte{0x05, 0x01, 0x00}, // SOCKS5, 1 method, no auth
			wantError: false,
			expected:  []byte{0x05, 0x00}, // SOCKS5, no auth selected
		},
		{
			name:      "Valid handshake multiple methods",
			input:     []byte{0x05, 0x03, 0x00, 0x01, 0x02}, // SOCKS5, 3 methods
			wantError: false,
			expected:  []byte{0x05, 0x00},
		},
		{
			name:      "Invalid SOCKS version",
			input:     []byte{0x04, 0x01, 0x00}, // SOCKS4
			wantError: true,
			expected:  nil,
		},
		{
			name:      "Incomplete header",
			input:     []byte{0x05}, // Only version
			wantError: true,
			expected:  nil,
		},
		{
			name:      "Incomplete methods",
			input:     []byte{0x05, 0x02, 0x00}, // Says 2 methods but only provides 1
			wantError: true,
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := NewMockConn()
			conn.AddReadData(tt.input)

			err := server.handleInitialHandshake(conn)
			if (err != nil) != tt.wantError {
				t.Errorf("handleInitialHandshake() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				written := conn.GetWrittenData()
				if !bytes.Equal(written, tt.expected) {
					t.Errorf("handleInitialHandshake() wrote %v, want %v", written, tt.expected)
				}
			}
		})
	}
}

func TestHandleRequest(t *testing.T) {
	cfg := &config.Config{
		Username:     "testuser",
		Password:     "testpass",
		UpstreamHost: "proxy.example.com",
		UpstreamPort: 1080,
	}
	server := NewServer(cfg)

	tests := []struct {
		name         string
		input        []byte
		wantError    bool
		expectedAddr string
	}{
		{
			name: "IPv4 request",
			input: []byte{
				0x05, 0x01, 0x00, 0x01, // SOCKS5, CONNECT, reserved, IPv4
				192, 168, 1, 1,          // IP: 192.168.1.1
				0x00, 0x50,              // Port: 80
			},
			wantError:    false,
			expectedAddr: "192.168.1.1:80",
		},
		{
			name: "Domain name request",
			input: []byte{
				0x05, 0x01, 0x00, 0x03,    // SOCKS5, CONNECT, reserved, domain
				0x0b,                       // Domain length: 11
				'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', // example.com
				0x01, 0xbb, // Port: 443
			},
			wantError:    false,
			expectedAddr: "example.com:443",
		},
		{
			name: "IPv6 request",
			input: []byte{
				0x05, 0x01, 0x00, 0x04, // SOCKS5, CONNECT, reserved, IPv6
				0x20, 0x01, 0x0d, 0xb8, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // IPv6 address
				0x00, 0x50, // Port: 80
			},
			wantError:    false,
			expectedAddr: "2001:db8::1:80",
		},
		{
			name: "Invalid SOCKS version",
			input: []byte{
				0x04, 0x01, 0x00, 0x01, // SOCKS4
				192, 168, 1, 1,
				0x00, 0x50,
			},
			wantError: true,
		},
		{
			name: "Unsupported address type",
			input: []byte{
				0x05, 0x01, 0x00, 0x05, // Invalid address type
				192, 168, 1, 1,
				0x00, 0x50,
			},
			wantError: true,
		},
		{
			name: "Incomplete request",
			input: []byte{
				0x05, 0x01, 0x00, // Missing address type and data
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := NewMockConn()
			conn.AddReadData(tt.input)

			addr, err := server.handleRequest(conn)
			if (err != nil) != tt.wantError {
				t.Errorf("handleRequest() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if addr != tt.expectedAddr {
					t.Errorf("handleRequest() addr = %v, want %v", addr, tt.expectedAddr)
				}

				// Check that success response was written
				written := conn.GetWrittenData()
				expected := []byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}
				if !bytes.Equal(written, expected) {
					t.Errorf("handleRequest() wrote %v, want %v", written, expected)
				}
			}
		})
	}
}

func TestForwardRequest(t *testing.T) {
	cfg := &config.Config{
		Username:     "testuser",
		Password:     "testpass",
		UpstreamHost: "proxy.example.com",
		UpstreamPort: 1080,
	}
	server := NewServer(cfg)

	tests := []struct {
		name       string
		target     string
		response   []byte
		wantError  bool
		wantOutput []byte
	}{
		{
			name:   "Successful forward",
			target: "example.com:80",
			response: []byte{
				0x05, 0x00, 0x00, 0x01, // Success response
				0, 0, 0, 0, 0, 0, // Bound address (ignored)
			},
			wantError: false,
			wantOutput: []byte{
				0x05, 0x01, 0x00, 0x03, // SOCKS5, CONNECT, reserved, domain
				0x0b,                    // Domain length: 11
				'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', // example.com
				0x00, 0x50, // Port: 80
			},
		},
		{
			name:   "Failed connection",
			target: "example.com:80",
			response: []byte{
				0x05, 0x01, 0x00, 0x01, // Connection refused
				0, 0, 0, 0, 0, 0,
			},
			wantError: true,
		},
		{
			name:      "Invalid target",
			target:    "invalid-target",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := NewMockConn()
			if tt.response != nil {
				conn.AddReadData(tt.response)
			}

			err := server.forwardRequest(conn, tt.target)
			if (err != nil) != tt.wantError {
				t.Errorf("forwardRequest() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.wantOutput != nil {
				written := conn.GetWrittenData()
				if !bytes.Equal(written, tt.wantOutput) {
					t.Errorf("forwardRequest() wrote %v, want %v", written, tt.wantOutput)
				}
			}
		})
	}
}

// MockTCPConn extends MockConn to implement CloseWrite
type MockTCPConn struct {
	*MockConn
	writeClosedChan chan bool
}

func NewMockTCPConn() *MockTCPConn {
	return &MockTCPConn{
		MockConn:        NewMockConn(),
		writeClosedChan: make(chan bool, 1),
	}
}

func (m *MockTCPConn) CloseWrite() error {
	select {
	case m.writeClosedChan <- true:
	default:
	}
	return nil
}

func TestForwardTraffic(t *testing.T) {
	cfg := &config.Config{
		Username:     "testuser",
		Password:     "testpass",
		UpstreamHost: "proxy.example.com",
		UpstreamPort: 1080,
	}
	server := NewServer(cfg)

	client := NewMockTCPConn()
	upstream := NewMockTCPConn()

	// Add test data
	clientData := []byte("client to upstream data")
	upstreamData := []byte("upstream to client data")

	client.AddReadData(clientData)
	upstream.AddReadData(upstreamData)

	// Start forwarding
	done := make(chan bool)
	go func() {
		server.forwardTraffic(client, upstream)
		done <- true
	}()

	// Wait a bit for data to be forwarded
	time.Sleep(10 * time.Millisecond)

	// Close the connections to stop forwarding
	client.Close()
	upstream.Close()

	// Wait for forwarding to complete
	select {
	case <-done:
		// Forwarding completed
	case <-time.After(time.Second):
		t.Error("Forwarding did not complete in time")
	}

	// Check that data was forwarded correctly
	upstreamWritten := upstream.GetWrittenData()
	if !bytes.Equal(upstreamWritten, clientData) {
		t.Errorf("Client data not forwarded to upstream: got %v, want %v", upstreamWritten, clientData)
	}

	clientWritten := client.GetWrittenData()
	if !bytes.Equal(clientWritten, upstreamData) {
		t.Errorf("Upstream data not forwarded to client: got %v, want %v", clientWritten, upstreamData)
	}
}

// Integration test with real TCP connections
func TestServerIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &config.Config{
		Username:     "testuser",
		Password:     "testpass",
		UpstreamHost: "127.0.0.1",
		UpstreamPort: 9999, // Use a port that doesn't exist
	}

	server := NewServer(cfg)

	// Start server on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	
	localAddr := listener.Addr().String()
	listener.Close()

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		err := server.Start(localAddr)
		if err != nil {
			errChan <- err
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Try to connect to the server
	conn, err := net.Dial("tcp", localAddr)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Send SOCKS5 handshake
	_, err = conn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		t.Fatalf("Failed to send handshake: %v", err)
	}

	// Read handshake response
	response := make([]byte, 2)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		t.Fatalf("Failed to read handshake response: %v", err)
	}

	expected := []byte{0x05, 0x00}
	if !bytes.Equal(response, expected) {
		t.Errorf("Unexpected handshake response: got %v, want %v", response, expected)
	}

	// Stop the server
	server.Stop()

	// Check if server stopped without errors
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(2 * time.Second):
		// Server stopped gracefully
	}
}

func TestServerConcurrentConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent connection test in short mode")
	}

	cfg := &config.Config{
		Username:     "testuser",
		Password:     "testpass",
		UpstreamHost: "127.0.0.1",
		UpstreamPort: 9999,
	}

	server := NewServer(cfg)

	// Start server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	
	localAddr := listener.Addr().String()
	listener.Close()

	go func() {
		server.Start(localAddr)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create multiple concurrent connections
	const numConnections = 10
	var wg sync.WaitGroup
	wg.Add(numConnections)

	for i := 0; i < numConnections; i++ {
		go func(id int) {
			defer wg.Done()

			conn, err := net.Dial("tcp", localAddr)
			if err != nil {
				t.Errorf("Connection %d failed: %v", id, err)
				return
			}
			defer conn.Close()

			// Send SOCKS5 handshake
			_, err = conn.Write([]byte{0x05, 0x01, 0x00})
			if err != nil {
				t.Errorf("Connection %d handshake write failed: %v", id, err)
				return
			}

			// Read response
			response := make([]byte, 2)
			_, err = io.ReadFull(conn, response)
			if err != nil {
				t.Errorf("Connection %d handshake read failed: %v", id, err)
				return
			}

			if !bytes.Equal(response, []byte{0x05, 0x00}) {
				t.Errorf("Connection %d unexpected response: %v", id, response)
			}
		}(i)
	}

	// Wait for all connections to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// All connections completed
	case <-time.After(5 * time.Second):
		t.Error("Concurrent connections test timed out")
	}

	// Stop server
	server.Stop()
}