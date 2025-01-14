package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error handling request: ", err.Error())
		os.Exit(1)
	}

	var response string
	lines := strings.Split(string(buf), "\r\n")
	method := strings.Split(lines[0], " ")[0]
	pathFile := strings.Split(lines[0], " ")[1]
	acceptEncoding := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "Accept-Encoding:") {
			acceptEncoding = strings.TrimSpace(strings.Split(line, ":")[1])
			break
		}
	}

	if method == "GET" && pathFile == "/" {
		response = "HTTP/1.1 200 OK\r\n\r\n"
	} else if method == "GET" && strings.HasPrefix(pathFile, "/echo/") {
		content := pathFile[6:]
		if strings.Contains(acceptEncoding, "gzip") {
			var buffer bytes.Buffer
			writer := gzip.NewWriter(&buffer)
			_, err := writer.Write([]byte(content))
			if err != nil {
				fmt.Println("Error writing gzip response: ", err.Error())
				os.Exit(1)
			}
			writer.Close()
			compressedContent := buffer.Bytes()
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Encoding: gzip\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(compressedContent), string(compressedContent))
		} else {
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(content), content)
		}
	} else if method == "GET" && pathFile == "/user-agent" {
		ua := strings.Split(lines[2], " ")[1]
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(ua), ua)
	} else if method == "GET" && strings.HasPrefix(pathFile, "/files/") {
		dir := os.Args[2]
		content, err := os.ReadFile(path.Join(dir, pathFile[7:]))
		if err != nil {
			response = "HTTP/1.1 404 Not Found\r\n\r\n"
		} else {
			if strings.Contains(acceptEncoding, "gzip") {
				var buffer bytes.Buffer
				writer := gzip.NewWriter(&buffer)
				_, err := writer.Write(content)
				if err != nil {
					fmt.Println("Error writing gzip response: ", err.Error())
					os.Exit(1)
				}
				writer.Close()
				compressedContent := buffer.Bytes()
				response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Encoding: gzip\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(compressedContent), string(compressedContent))
			} else {
				response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(content), string(content))
			}
		}
	} else if method == "POST" && strings.HasPrefix(pathFile, "/files/") {
		content := strings.Trim(lines[len(lines)-1], "\x00")
		dir := os.Args[2]
		_ = os.WriteFile(path.Join(dir, pathFile[7:]), []byte(content), 0644)
		response = "HTTP/1.1 201 Created\r\n\r\n"
	} else {
		response = "HTTP/1.1 404 Not Found\r\n\r\n"
	}

	conn.Write([]byte(response))
}
