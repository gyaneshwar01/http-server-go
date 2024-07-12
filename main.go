package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

const CRLF = "\r\n"

func main() {
	var dirFlag = flag.String("directory", "", "directory where files are stored")
	flag.Parse()

	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn, *dirFlag)
	}
}

func handleConnection(conn net.Conn, dirFlag string) {
	defer conn.Close()

	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading connection: ", err)
	}
	request := string(buf)

	lines := strings.Split(request, CRLF)
	path := strings.Split(lines[0], " ")[1]
	method := strings.Split(lines[0], " ")[0]

	headers := strings.Split(request, "\r\n")[1:]
	for _, header := range headers {
		fmt.Println(header)
	}

	// for i, line := range lines {
	// 	fmt.Printf("line %d: %s\n", i, line)
	// }

	var response string

	if path == "/" {
		response = "HTTP/1.1 200 OK\r\n\r\n"
	} else if strings.HasPrefix(path, "/echo/") {
		msg := path[6:]
		encoding := ""
		var encodings []string

		for _, header := range headers {
			if strings.Split(header, ": ")[0] == "Accept-Encoding" {
				encodings = strings.Split(strings.Split(header, ": ")[1], ",")
			}
		}

		for _, el := range encodings {
			if strings.TrimLeft(el, " ") == "gzip" {
				encoding = "gzip"
				break
			}
		}

		if encoding != "" {
			if encoding == "gzip" {
				var b bytes.Buffer
				w := gzip.NewWriter(&b)
				w.Write([]byte(msg))
				w.Close()

				response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Encoding: gzip\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", b.Len(), b.Bytes())
			} else {
				response = "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n"
			}
		} else {
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(msg), msg)
		}
	} else if path == "/user-agent" {
		userAgent := strings.Split(lines[2], " ")[1]
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(userAgent), userAgent)
	} else if strings.HasPrefix(path, "/files") && dirFlag != "" {
		filename := path[7:]
		fmt.Println(dirFlag + filename)
		if method == "GET" {
			if file, err := os.ReadFile(dirFlag + filename); err == nil {
				content := string(file)
				response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(content), content)
			} else {
				response = "HTTP/1.1 404 Not Found\r\n\r\n"
			}
		} else if method == "POST" {
			file := []byte(strings.Trim(lines[len(lines)-1], "\x00"))
			fmt.Printf("file_text: %s\n", strings.Trim(lines[len(lines)-1], "\x00"))
			if err := os.WriteFile(dirFlag+filename, file, 0644); err == nil {
				fmt.Println("wrote file")
				response = "HTTP/1.1 201 Created\r\n\r\n"
			} else {
				response = "HTTP/1.1 404 Not Found\r\n\r\n"
			}
		}
	} else {
		response = "HTTP/1.1 404 Not Found\r\n\r\n"
	}
	conn.Write([]byte(response))
}
