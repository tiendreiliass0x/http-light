package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

func main() {
	// Create a TCP listener
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error starting TCP listener: %v", err)
		os.Exit(1)
	}
	defer listener.Close()

	log.Println("Listening on :8080")

	for {
		// Accept a new connection
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// Handle the connection concurrently
		go handleConnection(conn)
	}
}

type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       string
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("Accepted connection from %v", conn.RemoteAddr())

	// Parse the HTTP request
	req, err := parseRequest(conn)
	if err != nil {
		log.Printf("Error parsing request: %v", err)
		writeErrorResponse(conn, 400, "Bad Request")
		return
	}

	// Generate a response
	res := handleRequest(req)

	// Write the response
	err = writeResponse(conn, res)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func parseRequest(conn net.Conn) (*Request, error) {
	reader := bufio.NewReader(conn)
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.TrimSpace(requestLine), " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed request line")
	}

	method := parts[0]
	url := parts[1]

	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		headerParts := strings.SplitN(line, ": ", 2)
		if len(headerParts) != 2 {
			return nil, fmt.Errorf("malformed header line")
		}
		headers[headerParts[0]] = headerParts[1]
	}

	var body string
	if contentLength, ok := headers["Content-Length"]; ok {
		length, err := strconv.Atoi(contentLength)
		if err != nil {
			return nil, fmt.Errorf("invalid Content-Length")
		}
		bodyBuffer := bufferPool.Get().([]byte)
		defer bufferPool.Put(bodyBuffer)
		bodyBytes := bodyBuffer[:length]
		_, err = io.ReadFull(reader, bodyBytes)
		if err != nil {
			return nil, err
		}
		body = string(bodyBytes)
	}

	return &Request{
		Method:  method,
		URL:     url,
		Headers: headers,
		Body:    body,
	}, nil
}

func handleRequest(req *Request) *Response {
	switch req.URL {
	case "/":
		return handleRoot(req)
	case "/hello":
		return handleHello(req)
	default:
		return &Response{
			StatusCode: 404,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "404 Not Found",
		}
	}
}

func handleRoot(req *Request) *Response {
	return &Response{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: "Welcome to the root page!",
	}
}

func handleHello(req *Request) *Response {
	name := req.Headers["Name"]
	if name == "" {
		name = "World"
	}
	return &Response{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: fmt.Sprintf("Hello, %s!", name),
	}
}

func writeResponse(conn net.Conn, res *Response) error {
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", res.StatusCode, statusText(res.StatusCode))
	_, err := conn.Write([]byte(statusLine))
	if err != nil {
		return err
	}

	for key, value := range res.Headers {
		headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
		_, err := conn.Write([]byte(headerLine))
		if err != nil {
			return err
		}
	}

	_, err = conn.Write([]byte("\r\n"))
	if err != nil {
		return err
	}

	_, err = conn.Write([]byte(res.Body))
	return err
}

func writeErrorResponse(conn net.Conn, statusCode int, message string) {
	res := &Response{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: message,
	}
	err := writeResponse(conn, res)
	if err != nil {
		log.Printf("Error writing error response: %v", err)
	}
}

func statusText(statusCode int) string {
	switch statusCode {
	case 200:
		return "OK"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	default:
		return "Unknown Status"
	}
}
