package hostinfo

import (
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	once   sync.Once
	cached string
)

var echoServices = []string{
	"https://api.ipify.org",
	"https://icanhazip.com",
	"https://ifconfig.me/ip",
}

func PublicIP() string {
	once.Do(func() { cached = detect() })
	return cached
}

func PublicIPDashed() string {
	ip := net.ParseIP(PublicIP())
	if ip == nil || ip.To4() == nil {
		return strings.ReplaceAll(PublicIP(), ":", "-")
	}
	return strings.ReplaceAll(PublicIP(), ".", "-")
}

func detect() string {
	if v := strings.TrimSpace(os.Getenv("AETHER_PUBLIC_HOST")); v != "" {
		return v
	}
	if v := echoIP(); v != "" {
		return v
	}
	return dialIP()
}

func echoIP() string {
	client := &http.Client{Timeout: 2 * time.Second}
	for _, url := range echoServices {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		ip := net.ParseIP(strings.TrimSpace(string(body)))
		if ip != nil {
			return ip.String()
		}
	}
	return ""
}

func dialIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "127.0.0.1"
	}
	if ip := addr.IP.String(); ip != "" {
		return ip
	}
	return "127.0.0.1"
}
