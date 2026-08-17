package notifications

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

type resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

// TargetPolicy controls which network destinations notification providers may contact.
// Private addresses are denied unless explicitly enabled or the hostname is allowlisted.
type TargetPolicy struct {
	AllowPrivate bool
	AllowedHosts []string
	resolver     resolver
	dialer       *net.Dialer
}

func NewTargetPolicy(allowPrivate bool, allowedHosts []string) TargetPolicy {
	return TargetPolicy{AllowPrivate: allowPrivate, AllowedHosts: allowedHosts}
}

func (p TargetPolicy) client() *http.Client {
	if p.resolver == nil {
		p.resolver = net.DefaultResolver
	}
	if p.dialer == nil {
		p.dialer = &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           p.dialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	return &http.Client{Transport: transport, Timeout: 15 * time.Second, CheckRedirect: p.checkRedirect}
}

func (p TargetPolicy) ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return fmt.Errorf("provider URL must be an absolute HTTP(S) URL")
	}
	if u.User != nil {
		return fmt.Errorf("provider URL must not contain credentials")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !p.permits(u.Hostname(), ip) {
		return fmt.Errorf("notification target address is not permitted")
	}
	return nil
}

func (p TargetPolicy) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid notification target: %w", err)
	}
	addresses, err := p.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve notification target: %w", err)
	}
	for _, address := range addresses {
		if !p.permits(host, address.IP) {
			continue
		}
		conn, dialErr := p.dialer.DialContext(ctx, network, net.JoinHostPort(address.IP.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		err = dialErr
	}
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("notification target resolves only to blocked addresses")
}

func (p TargetPolicy) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 3 {
		return fmt.Errorf("notification redirect limit exceeded")
	}
	return p.ValidateURL(req.URL.String())
}

func (p TargetPolicy) permits(host string, ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if addr.IsUnspecified() || addr.IsMulticast() {
		return false
	}
	if p.AllowPrivate || p.allowedHost(host) {
		return true
	}
	return addr.IsGlobalUnicast() && !addr.IsPrivate() && !addr.IsLoopback() && !addr.IsLinkLocalUnicast() && !isCarrierGradeNAT(addr)
}

func (p TargetPolicy) allowedHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	for _, allowed := range p.AllowedHosts {
		if host == strings.TrimSuffix(strings.ToLower(strings.TrimSpace(allowed)), ".") {
			return true
		}
	}
	return false
}

func isCarrierGradeNAT(addr netip.Addr) bool {
	prefix := netip.MustParsePrefix("100.64.0.0/10")
	return addr.Is4() && prefix.Contains(addr)
}
