package main

import (
	"context"
	"net"
	"net/http"
	"time"
)

type NetStatus struct {
	Online   bool   `json:"online"`
	PingOK   bool   `json:"ping_ok"`
	DNSOK    bool   `json:"dns_ok"`
	HTTPOK   bool   `json:"http_ok"`
	CheckedAt string `json:"checked_at"`
}

// checkNetwork faz 3 verificações com timeout curto.
// Não bloqueia o fluxo se falhar — apenas registra o estado.
func checkNetwork() NetStatus {
	status := NetStatus{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// 1. Ping (TCP na verdade — ICMP precisa de root e é lento)
	conn, err := net.DialTimeout("tcp", "1.1.1.1:53", 2*time.Second)
	if err == nil {
		status.PingOK = true
		conn.Close()
	}

	// 2. DNS
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := net.DefaultResolver.LookupHost(ctx, "one.one.one.one"); err == nil {
		status.DNSOK = true
	}

	// 3. HTTP
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://connectivitycheck.gstatic.com/generate_204")
	if err == nil {
		status.HTTPOK = true
		resp.Body.Close()
	}

	status.Online = status.PingOK && status.DNSOK && status.HTTPOK
	return status
}