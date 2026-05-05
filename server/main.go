package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

var agentSub map[string]net.Conn

const (
	TypeAUTH      byte = 0x01
	TypeOpen      byte = 0x02
	TypeDATA      byte = 0x03
	TypeCLOSE     byte = 0x04
	TypePING      byte = 0x05
	TypePONG      byte = 0x06
	TypeERROR     byte = 0x07
	TypeCONTROL   byte = 0x08
	TypeSTRAMINFO byte = 0x09
)

// type Flag,
const (
	FlagCompressed  byte = 0b00000001
	FlagEncrypted   byte = 0b00000010
	FlagPriority    byte = 0b00000100
	FlagACKRequired byte = 0b00001000
)
const Version byte = 0x01

type Frame struct {
	Version  byte
	Type     byte
	Flags    byte
	StreamID uint32
	Data     []byte
}

const HeaderSize = 11

func main() {
	// TCP listener yaratamiz
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("TCP server 8080-portda ishga tushdi...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	// 1. Kelayotgan ma'lumotni o'qiymiz
	reader := bufio.NewReader(conn)

	// HTTP sarlavhalarining birinchi qatorini o'qiymiz (Masalan: GET / HTTP/1.1)
	firstLine, _ := reader.ReadString('\n')
	if firstLine == "" {
		return
	}

	// 2. "Host:" sarlavhasini qidiramiz
	var host string
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" { // Sarlavhalar tugadi
			break
		}
		if strings.HasPrefix(line, "Host:") {
			host = strings.TrimSpace(strings.TrimPrefix(line, "Host:"))
			break
		}
	}

	// 3. Subdomenni ajratamiz
	if host != "" {
		// Portni olib tashlash (ali.localhost:8080 -> ali.localhost)
		h, _, err := net.SplitHostPort(host)
		if err != nil {
			h = host
		}

		parts := strings.Split(h, ".")
		if len(parts) >= 2 && parts[len(parts)-1] == "localhost" {
			subdomain := parts[0]

			log.Printf("TCP orqali kelgan subdomen: %s", subdomain)

			// 4. HTTP javob qaytaramiz (TCP orqali qo'lda yoziladi)
			response := fmt.Sprintf(
				"HTTP/1.1 200 OK\r\n"+
					"Content-Type: text/plain; charset=utf-8\r\n"+
					"Content-Length: %d\r\n"+
					"\r\n"+
					"Siz TCP orqali bog'landingiz. Subdomen: %s",
				len(subdomain)+45, subdomain)

			conn.Write([]byte(response))
			return
		}
	}

	// Agar subdomen topilmasa
	conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\nHost topilmadi."))
}
