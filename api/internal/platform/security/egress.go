package security

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrUnsafeOutboundURL = errors.New("unsafe outbound URL")

type EgressPolicy struct {
	AllowLoopback bool
}

func ValidateOutboundURL(raw string) error {
	return validateOutboundURL(raw, EgressPolicy{}, true)
}

func ValidateOutboundURLForTest(raw string) error {
	return validateOutboundURL(raw, EgressPolicy{AllowLoopback: true}, true)
}

func ValidateOutboundURLSyntax(raw string) error {
	return validateOutboundURL(raw, EgressPolicy{}, false)
}

func NewHTTPClient(timeout time.Duration) *http.Client {
	return newHTTPClient(timeout, EgressPolicy{})
}

func NewTestHTTPClient(timeout time.Duration) *http.Client {
	return newHTTPClient(timeout, EgressPolicy{AllowLoopback: true})
}

func newHTTPClient(timeout time.Duration, policy EgressPolicy) *http.Client {
	transport := &http.Transport{
		Proxy:       nil,
		DialContext: (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	}
	transport.DialContext = dialContext(policy)
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return ErrUnsafeOutboundURL
			}
			return validateOutboundURL(req.URL.String(), policy, true)
		},
	}
}

func dialContext(policy EgressPolicy) func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			host = address
			port = ""
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", strings.Trim(host, "[]"))
		if err != nil {
			return nil, ErrUnsafeOutboundURL
		}
		for _, ip := range ips {
			if isBlockedIP(ip, policy) {
				continue
			}
			target := net.JoinHostPort(ip.String(), port)
			return dialer.DialContext(ctx, network, target)
		}
		return nil, ErrUnsafeOutboundURL
	}
}

func validateOutboundURL(raw string, policy EgressPolicy, resolve bool) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u == nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return ErrUnsafeOutboundURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrUnsafeOutboundURL
	}
	if port := u.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return ErrUnsafeOutboundURL
		}
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || isDockerHost(host) {
		return ErrUnsafeOutboundURL
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip, policy) {
			return ErrUnsafeOutboundURL
		}
		return nil
	}
	if !resolve {
		return nil
	}
	resolveContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIP(resolveContext, "ip", host)
	if err != nil {
		return fmt.Errorf("%w: resolve host", ErrUnsafeOutboundURL)
	}
	for _, ip := range ips {
		if isBlockedIP(ip, policy) {
			return ErrUnsafeOutboundURL
		}
	}
	return nil
}

func isDockerHost(host string) bool {
	switch host {
	case "host.docker.internal", "gateway.docker.internal", "host.containers.internal":
		return true
	default:
		return false
	}
}

func isBlockedIP(ip net.IP, policy EgressPolicy) bool {
	if policy.AllowLoopback && ip.IsLoopback() {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
		return true
	}
	return false
}
