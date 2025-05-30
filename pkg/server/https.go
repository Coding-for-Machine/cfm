// pkg/server/https.go
package server

import (
	"crypto/tls"
	"fmt"
	"log"

	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/acme/autocert"
)

type HTTPSServer struct {
	server      *fasthttp.Server
	certManager *autocert.Manager
	domain      string
}

func NewHTTPSServer(domain string, handler fasthttp.RequestHandler) *HTTPSServer {
	certManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain, "*."+domain),
		Cache:      autocert.DirCache("./certs"),
	}

	return &HTTPSServer{
		server: &fasthttp.Server{
			Handler: handler,
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
				NextProtos:     []string{"h2", "http/1.1"},
			},
		},
		certManager: certManager,
		domain:      domain,
	}
}

func (s *HTTPSServer) ListenAndServe() error {
	// HTTP redirect server
	go func() {
		redirectServer := &fasthttp.Server{
			Handler: func(ctx *fasthttp.RequestCtx) {
				host := string(ctx.Host())
				url := fmt.Sprintf("https://%s%s", host, ctx.RequestURI())
				ctx.Redirect(url, fasthttp.StatusMovedPermanently)
			},
		}

		log.Printf("🔄 HTTP redirect server starting on :80")
		if err := redirectServer.ListenAndServe(":80"); err != nil {
			log.Printf("HTTP redirect server error: %v", err)
		}
	}()

	// ACME HTTP-01 challenge server
	go func() {
		challengeServer := &fasthttp.Server{
			Handler: s.certManager.HTTPHandler(nil),
		}

		if err := challengeServer.ListenAndServe(":80"); err != nil {
			log.Printf("ACME challenge server error: %v", err)
		}
	}()

	log.Printf("🔒 HTTPS server starting on :443 for domain: %s", s.domain)
	return s.server.ListenAndServeTLS(":443", "", "")
}
