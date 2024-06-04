package main

import (
	"io"
	"io/ioutil"
	"net"
	"strings"
	"testing"
	"time"
)

func TestParseRequest(t *testing.T) {
	requestString := "GET /hello HTTP/1.1\r\nHost: localhost\r\nName: Go\r\n\r\n"
	conn := &mockConn{reader: strings.NewReader(requestString)}
	req, err := parseRequest(conn)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if req.Method != "GET" || req.URL != "/hello" {
		t.Fatalf("Unexpected request: %+v", req)
	}

	if req.Headers["Host"] != "localhost" || req.Headers["Name"] != "Go" {
		t.Fatalf("Unexpected headers: %+v", req.Headers)
	}
}

func TestWriteResponse(t *testing.T) {
	conn := &mockConn{writer: &strings.Builder{}}
	res := &Response{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: "Hello, Go!",
	}

	err := writeResponse(conn, res)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	responseString := conn.writer.(*strings.Builder).String()
	if !strings.Contains(responseString, "HTTP/1.1 200 OK") ||
		!strings.Contains(responseString, "Content-Type: text/plain") ||
		!strings.Contains(responseString, "Hello, Go!") {
		t.Fatalf("Unexpected response: %s", responseString)
	}
}

type mockConn struct {
	reader io.Reader
	writer io.Writer
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.reader.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writer.Write(b)
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return nil
}

func (m *mockConn) RemoteAddr() net.Addr {
	return nil
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestServer(t *testing.T) {
	go main()

	tests := []struct {
		request  string
		expected string
	}{
		{"GET / HTTP/1.1\r\n\r\n", "HTTP/1.1 200 OK\r\n"},
		{"GET /hello HTTP/1.1\r\nName: Go\r\n\r\n", "HTTP/1.1 200 OK\r\nHello, Go!"},
		{"GET /notfound HTTP/1.1\r\n\r\n", "HTTP/1.1 404 Not Found\r\n"},
	}

	for _, test := range tests {
		conn, err := net.Dial("tcp", "localhost:8080")
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		_, err = conn.Write([]byte(test.request))
		if err != nil {
			t.Fatalf("Failed to write request: %v", err)
		}

		response, err := ioutil.ReadAll(conn)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		if !strings.Contains(string(response), test.expected) {
			t.Fatalf("Unexpected response: %s", response)
		}

		conn.Close()
	}
}
