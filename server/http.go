package main

import (
	"fmt"
	"strings"
)

type ServerHttp struct {
	Request
	Response
}
type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Body    string
}
type Response struct {
	// start line
	Version    string
	StatusCode map[uint16]string
	// headers
	Headers map[string]string
	// body
	body string
}

var Status = map[int16]string{
	// Axborotga oid javoblar
	100: "Continue",
	101: "Switching Protocols",
	102: "Processing",
	103: "Early Hints",
	// Muvaffaqiyatli javoblar
	200: "OK",
	201: "Created",
	202: "Accepted",
	203: "Non-Authoritative Information",
	204: "No Content",
	205: "Reset Content",
	206: "Partial Content",
	207: "Multi-Status",
	208: "Already Reported",
	226: "IM Used",
	// Yo'naltirish xabarlari
	300: "Multiple Choices",
	301: "Moved Permanently",
	302: "Found",
	303: "See Other",
	304: "Not Modified",
	305: "Use Proxy",
	306: "unused",
	307: "Temporary Redirect",
	308: "Permanent Redirect",
	// Mijoz xatolariga javoblar
	400: "Bad Request",
	401: "Unauthorized",
	402: "Payment Required",
	403: "Forbidden",
	404: "Not Found",
	405: "Method Not Allowed",
	406: "Not Acceptable",
	407: "Proxy Authentication Required",
	408: "Request Timeout",
	409: "Conflict",
	410: "Gone",
	411: "Length Required",
	412: "Precondition Failed",
	413: "Content Too Large",
	414: "URI Too Long",
	415: "Unsupported Media Type",
	416: "Range Not Satisfiable",
	417: "Expectation Failed",
	418: "I'm a teapot",
	421: "Misdirected Request",
	422: "Unprocessable Content",
	423: "Locked",
	424: "Failed Dependency",
	425: "Too Early",
	426: "Upgrade Required",
	428: "Precondition Required",
	431: "Request Header Fields Too Large",
	451: "Unavailable For Legal Reasons",
	// Server xatolariga javoblar
	500: "Internal Server Error",
	501: "Not Implemented",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Timeout",
	505: "HTTP Version Not Supported",
	506: "Variant Also Negotiates",
	507: "Insufficient Storage",
	508: "Loop Detected",
	510: "Not Extended",
	511: "Network Authentication Required",
}

func (s *ServerHttp) RequestHandler(message string) error {
	// 1. So'rovni Header va Body qismlariga ajratamiz (\r\n\r\n bo'yicha)
	parts := strings.SplitN(message, "\r\n\r\n", 2)
	headerSection := parts[0]
	bodySection := ""
	if len(parts) > 1 {
		bodySection = parts[1]
	}

	// 2. Header qismini qatorlarga bo'lamiz
	headerLines := strings.Split(headerSection, "\r\n")
	if len(headerLines) == 0 {
		return fmt.Errorf("bo'sh so'rov")
	}

	// 3. Start Line tahlili (Method Path Version)
	startLineParts := strings.Fields(headerLines[0])
	if len(startLineParts) != 3 {
		return fmt.Errorf("bad request: start line noto'g'ri")
	}

	// 4. Headersni Map ko'rinishiga keltirish
	headerMap := make(map[string]string)
	for i := 1; i < len(headerLines); i++ {
		line := headerLines[i]
		if line == "" {
			continue
		}
		kv := strings.SplitN(line, ": ", 2)
		if len(kv) == 2 {
			headerMap[kv[0]] = kv[1]
		}
	}

	// 5. ServerHttp ichidagi Request obyektini yangilash
	s.Request = Request{
		Method:  startLineParts[0],
		Path:    startLineParts[1],
		Version: startLineParts[2],
		Headers: headerMap,
		Body:    bodySection,
	}
	fmt.Printf("Metod: %s, Path: %s, Host:", s.Method, s.Path)
	return nil
}

// Response yaratish funksiyasi
func (s *ServerHttp) ResponseBuilder(statusCode int16, body string) string {
	// Status kodiga qarab matnni olamiz (masalan: 200 -> OK)
	statusText := Status[statusCode]
	if statusText == "" {
		statusText = "Unknown Status"
	}

	// Sarlavhalarni tayyorlaymiz
	s.Response.Headers = make(map[string]string)
	s.Response.Headers["Content-Type"] = "text/html; charset=UTF-8"
	s.Response.Headers["Content-Length"] = fmt.Sprintf("%d", len(body))
	s.Response.Headers["Connection"] = "close"

	// Start line: HTTP/1.1 200 OK
	res := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusText)

	// Headersni stringga aylantiramiz
	for key, value := range s.Response.Headers {
		res += fmt.Sprintf("%s: %s\r\n", key, value)
	}

	// Bo'sh qator va Body
	res += "\r\n" + body

	return res
}
