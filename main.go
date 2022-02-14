package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
)

const address = "0.0.0.0:8080"

func main() {
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalln("error listening:", err.Error())
	}
	defer l.Close()

	log.Println("listening on", address)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalln("error accepting connection:", err.Error())
		}
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	received, err := executeProxiedRequest(conn)
	if err != nil {
		log.Println("error executing proxied request:", err)
		return
	}
	log.Printf("Sending:\n%v\n\n", string(received))
	_, err = conn.Write([]byte(received))
	if err != nil {
		log.Println("error sending response to proxy client:", err)
		return
	}
}

func readConn(conn net.Conn) ([]byte, error) {
	var buf []byte
	for {
		tmp := make([]byte, 256)
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
				return nil, err
			}
			break
		}
		buf = append(buf, tmp[:n]...)
		if n < len(tmp) {
			break
		}
	}
	return buf, nil
}

func executeProxiedRequest(conn net.Conn) ([]byte, error) {
	buf, err := readConn(conn)
	if err != nil {
		return nil, err
	}
	log.Printf("Received:\n%v\n\n", string(buf))
	lines := strings.Split(string(buf), "\n")
	urlString := strings.Split(lines[0], " ")[1]
	URL, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	proxiedHostname := URL.Hostname()
	proxiedPort := URL.Port()
	if proxiedPort == "" {
		proxiedPort = "80"
	}
	lines[0] = strings.Replace(lines[0], urlString, URL.Path, 1)

	badHeaderIndex := -1
	for i, v := range lines {
		if strings.Contains(v, "Proxy-Connection") {
			badHeaderIndex = i
			break
		}
	}
	if badHeaderIndex != -1 {
		copy(lines[badHeaderIndex:], lines[badHeaderIndex+1:])
		lines = lines[:len(lines)-1]
	}

	toBeSent := []byte(strings.Join(lines, "\n"))

	proxiedConn, err := net.Dial("tcp", proxiedHostname+":"+proxiedPort)
	if err != nil {
		return nil, err
	}
	defer proxiedConn.Close()

	proxiedConn.Write(toBeSent)

	return readConn(proxiedConn)
}
