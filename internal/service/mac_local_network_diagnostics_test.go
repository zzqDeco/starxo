package service

import (
	"errors"
	"net"
	"testing"
)

func TestIsDiagnosticLocalNetworkHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "private ipv4", host: "192.168.31.59", want: true},
		{name: "loopback with port", host: "127.0.0.1:22", want: true},
		{name: "bonjour", host: "starxo.local", want: true},
		{name: "carrier grade nat", host: "100.64.1.2", want: true},
		{name: "public ipv4", host: "8.8.8.8", want: false},
		{name: "public dns name", host: "example.com", want: false},
		{name: "ipv6 loopback", host: "[::1]:22", want: true},
		{name: "ipv6 ula", host: "fd00::1", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDiagnosticLocalNetworkHost(tt.host); got != tt.want {
				t.Fatalf("isDiagnosticLocalNetworkHost(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestIsMacLocalNetworkPermissionLikeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "no route", err: errors.New("dial tcp 192.168.31.59:22: connect: no route to host"), want: true},
		{name: "unreachable", err: errors.New("network is unreachable"), want: true},
		{name: "operation not permitted", err: errors.New("operation not permitted"), want: true},
		{name: "refused", err: errors.New("connection refused"), want: false},
		{name: "nil", err: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMacLocalNetworkPermissionLikeError(tt.err); got != tt.want {
				t.Fatalf("isMacLocalNetworkPermissionLikeError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsDiagnosticLocalIP(t *testing.T) {
	for _, raw := range []string{"0.0.0.0", "169.254.1.1", "224.0.0.1", "fe80::1", "ff02::1"} {
		ip := net.ParseIP(raw)
		if ip == nil {
			t.Fatalf("invalid test IP %s", raw)
		}
		if !isDiagnosticLocalIP(ip) {
			t.Fatalf("expected %s to be treated as local/non-public", raw)
		}
	}
}
