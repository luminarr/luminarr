// Package safedialer provides HTTP transports that protect against SSRF attacks
// by filtering outbound connections to sensitive network addresses.
//
// Two transports are provided:
//   - Transport() — strict; blocks loopback, RFC-1918, link-local, CGNAT.
//     Use for indexers, download clients, webhooks, and torrent file fetches
//     where the target should be a public internet host.
//   - LANTransport() — permissive; only blocks cloud-metadata link-local ranges.
//     Use for the Radarr importer, which legitimately connects to an internal
//     host (localhost or LAN) that the user controls.
package safedialer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// blockedCIDRs is the list of address ranges that must never be reachable via
// user-supplied URLs. Covers: loopback, private, link-local, and unspecified.
var blockedCIDRs = func() []*net.IPNet {
	cidrs := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"::1/128",        // IPv6 loopback
		"10.0.0.0/8",     // RFC-1918 private
		"172.16.0.0/12",  // RFC-1918 private
		"192.168.0.0/16", // RFC-1918 private
		"169.254.0.0/16", // IPv4 link-local (AWS/GCP metadata: 169.254.169.254)
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique-local
		"0.0.0.0/8",      // "this" network
		"100.64.0.0/10",  // CGNAT / Tailscale
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, ipnet, err := net.ParseCIDR(c)
		if err != nil {
			panic("safedialer: invalid CIDR " + c + ": " + err.Error())
		}
		nets = append(nets, ipnet)
	}
	return nets
}()

// lanBlockedCIDRs only blocks the genuinely dangerous cloud-metadata ranges.
// RFC-1918 and loopback are allowed because the Radarr importer needs to reach
// an internal host (e.g. http://localhost:7878 or http://192.168.1.100:7878).
var lanBlockedCIDRs = func() []*net.IPNet {
	cidrs := []string{
		"169.254.0.0/16", // AWS/GCP/Azure instance metadata (169.254.169.254)
		"fe80::/10",      // IPv6 link-local
		"100.64.0.0/10",  // CGNAT / Tailscale
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, ipnet, err := net.ParseCIDR(c)
		if err != nil {
			panic("safedialer: invalid CIDR " + c + ": " + err.Error())
		}
		nets = append(nets, ipnet)
	}
	return nets
}()

// isBlocked returns true if ip falls within any of the blocked ranges.
func isBlocked(ip net.IP) bool {
	for _, cidr := range blockedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// isLANBlocked returns true if ip falls within the metadata-only blocked ranges.
func isLANBlocked(ip net.IP) bool {
	for _, cidr := range lanBlockedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// dialContext is a net.Dialer.DialContext replacement that resolves the host
// and rejects any resulting IP that falls within a blocked range.
func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("safedialer: parsing address %q: %w", addr, err)
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("safedialer: resolving %q: %w", host, err)
	}

	for _, a := range addrs {
		ip := net.ParseIP(a)
		if ip == nil {
			return nil, fmt.Errorf("safedialer: could not parse resolved IP %q", a)
		}
		if isBlocked(ip) {
			return nil, fmt.Errorf("safedialer: connection to %s (%s) is not allowed", host, ip)
		}
	}

	// All resolved addresses are public — proceed with the standard dialer.
	d := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return d.DialContext(ctx, network, net.JoinHostPort(addrs[0], port))
}

// Transport returns an *http.Transport that blocks requests to private/internal
// network addresses. Use this in place of http.DefaultTransport for all HTTP
// clients that connect to user-supplied URLs.
func Transport() *http.Transport {
	return &http.Transport{
		DialContext:           dialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       60 * time.Second,
	}
}

// lanDialContext is like dialContext but uses lanBlockedCIDRs, allowing
// private/loopback addresses (for Radarr import) while still blocking
// cloud-metadata endpoints.
func lanDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("safedialer: parsing address %q: %w", addr, err)
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("safedialer: resolving %q: %w", host, err)
	}

	for _, a := range addrs {
		ip := net.ParseIP(a)
		if ip == nil {
			return nil, fmt.Errorf("safedialer: could not parse resolved IP %q", a)
		}
		if isLANBlocked(ip) {
			return nil, fmt.Errorf("safedialer: connection to %s (%s) is not allowed", host, ip)
		}
	}

	d := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return d.DialContext(ctx, network, net.JoinHostPort(addrs[0], port))
}

// LANTransport returns an *http.Transport suitable for connecting to
// user-configured internal services (e.g. a Radarr instance on the LAN).
// It allows private/loopback addresses but still blocks cloud-metadata
// link-local ranges (169.254.0.0/16, fe80::/10, 100.64.0.0/10).
func LANTransport() *http.Transport {
	return &http.Transport{
		DialContext:           lanDialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       60 * time.Second,
	}
}
