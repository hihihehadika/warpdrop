package network

import (
	"errors"
	"net"
)

// ErrNoIPFound returned when no valid non-loopback local IPv4 address could be resolved.
var ErrNoIPFound = errors.New("no active local IPv4 interface found")

// GetLocalIP attempts to find the primary outbound local IPv4 address.
// It first tries a connectionless UDP dial to a public address to determine
// the active routing interface without sending real packets.
// If offline, it falls back to scanning all available network interfaces.
func GetLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
		if ok && localAddr.IP != nil && !localAddr.IP.IsLoopback() {
			ip4 := localAddr.IP.To4()
			if ip4 != nil {
				return ip4.String(), nil
			}
		}
	}

	return getFallbackIP()
}

// getFallbackIP scans local network interfaces and selects the first
// active, non-loopback, private IPv4 address.
func getFallbackIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip4 := ip.To4()
			if ip4 != nil {
				return ip4.String(), nil
			}
		}
	}

	return "", ErrNoIPFound
}
