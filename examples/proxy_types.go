package main

import (
	"log"

	"github.com/farshadnobody/Rubingo/rubingo"
)

func main() {
	// ═══════════════════════════════════════════════════════
	// HTTP Proxy
	// ═══════════════════════════════════════════════════════
	clientHTTP, _ := rubingo.NewClient("session_http",
		rubingo.WithProxy("http://user:pass@gate.decodo.com:10001"),
		rubingo.WithProxyEnabled(true),
	)

	// ═══════════════════════════════════════════════════════
	// HTTP Proxy با احراز هویت
	// ═══════════════════════════════════════════════════════
	clientHTTPAuth, _ := rubingo.NewClient("session_http_auth",
		rubingo.WithProxy("http://username:password@proxy.example.com:8080"),
		rubingo.WithProxyEnabled(true),
	)

	// ═══════════════════════════════════════════════════════
	// SOCKS5 Proxy
	// ═══════════════════════════════════════════════════════
	clientSOCKS, _ := rubingo.NewClient("session_socks",
		rubingo.WithProxy("socks5://127.0.0.1:1080"),
		rubingo.WithProxyEnabled(true),
	)

	// ═══════════════════════════════════════════════════════
	// SOCKS5 با احراز هویت
	// ═══════════════════════════════════════════════════════
	clientSOCKSAuth, _ := rubingo.NewClient("session_socks_auth",
		rubingo.WithProxy("socks5://user:pass@proxy.example.com:1080"),
		rubingo.WithProxyEnabled(true),
	)

	log.Println("HTTP Client:", clientHTTP)
	log.Println("HTTP Auth Client:", clientHTTPAuth)
	log.Println("SOCKS5 Client:", clientSOCKS)
	log.Println("SOCKS5 Auth Client:", clientSOCKSAuth)
}
