package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
)

const (
	Version  byte = 0x01
	TypeAUTH byte = 0x01
	TypeDATA byte = 0x03
)

func main() {
	subdomain := "ali" // Serverda ro'yxatdan o'tmoqchi bo'lgan nomimiz
	serverAddr := "localhost:8080"
	localAppAddr := "localhost:8000" // O'zimizda ishlab turgan loyiha porti

	// 1. Serverga ulanish
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Fatal("Serverga ulanib bo'lmadi:", err)
	}
	defer conn.Close()

	// 2. AUTH Frame yuborish
	// [Version(1b) | Type(1b) | Flags(1b) | StreamID(4b) | DataLen(4b) | Data(n)]
	authFrame := make([]byte, 11+len(subdomain))
	authFrame[0] = Version
	authFrame[1] = TypeAUTH
	binary.BigEndian.PutUint32(authFrame[3:7], 0) // StreamID hozircha 0
	binary.BigEndian.PutUint32(authFrame[7:11], uint32(len(subdomain)))
	copy(authFrame[11:], subdomain)

	conn.Write(authFrame)
	log.Printf("Agent [%s] bo'lib ro'yxatdan o'tdi. Server: %s", subdomain, serverAddr)

	// 3. Serverdan keladigan ma'lumotlarni kutish (Forwarding)
	for {
		// Bu yerda serverdan kelayotgan so'rovni o'qib, lokal 3000-portga yo'naltirish kerak
		// Soddalik uchun hozircha to'g'ridan-to'g'ri bog'laymiz:

		localConn, err := net.Dial("tcp", localAppAddr)
		if err != nil {
			log.Println("Lokal dastur (3000) o'chiq bo'lishi mumkin.")
			continue
		}

		log.Println("Yangi so'rov keldi, lokal serverga yo'naltirildi.")

		// Ma'lumotlarni ko'prik qilish
		go func() {
			defer localConn.Close()
			io.Copy(localConn, conn) // Serverdan kelganini lokalga
		}()
		io.Copy(conn, localConn) // Lokaldan kelganini serverga (brauzerga)
	}
}
