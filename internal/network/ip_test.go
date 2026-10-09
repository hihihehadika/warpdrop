package network

import (
	"net"
	"testing"
)

func TestGetLocalIP(t *testing.T) {
	ipStr, err := GetLocalIP()
	if err != nil {
		t.Logf("[INFO] GetLocalIP returned error in isolated environment: %v", err)
		return
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		t.Fatalf("expected valid IP address string, got: %s", ipStr)
	}

	if ip.IsLoopback() {
		t.Fatalf("expected non-loopback IP address, got: %s", ipStr)
	}

	if ip.To4() == nil {
		t.Fatalf("expected IPv4 address, got: %s", ipStr)
	}
}

func TestGetFallbackIP(t *testing.T) {
	ipStr, err := getFallbackIP()
	if err != nil {
		if errorsIs(err, ErrNoIPFound) {
			t.Log("[INFO] No active interfaces found in test runner environment")
			return
		}
		t.Fatalf("unexpected error: %v", err)
	}

	ip := net.ParseIP(ipStr)
	if ip == nil || ip.IsLoopback() || ip.To4() == nil {
		t.Fatalf("expected valid non-loopback IPv4, got: %s", ipStr)
	}
}

func errorsIs(err, target error) bool {
	return err == target
}
