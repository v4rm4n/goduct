// tunnel/netutils.go
package tunnel

import (
	"fmt"
	"net"
	"strings"
)

// ResolveHostPort takes a string like "eth0:8080", "localhost:80", or "192.168.1.5:443"
// and resolves the host portion to a valid IP address.
func ResolveHostPort(hostport string) (string, error) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", fmt.Errorf("invalid format %q (expected host:port): %w", hostport, err)
	}

	ip, err := resolveHost(host)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(ip, port), nil
}

func resolveHost(name string) (string, error) {
	if name == "" || name == "0.0.0.0" || name == "*" {
		return "0.0.0.0", nil
	}
	if name == "localhost" {
		return "127.0.0.1", nil
	}

	// 1. Is it already a valid IP?
	if ip := net.ParseIP(name); ip != nil {
		return ip.String(), nil
	}

	// 2. Is it a network interface? (e.g., eth0, wlan0, Ethernet, Wi-Fi)
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, i := range ifaces {
			if strings.EqualFold(i.Name, name) {
				addrs, err := i.Addrs()
				if err != nil {
					continue
				}
				for _, a := range addrs {
					// Prefer IPv4
					var ip net.IP
					switch v := a.(type) {
					case *net.IPNet:
						ip = v.IP
					case *net.IPAddr:
						ip = v.IP
					}
					if ip != nil && ip.To4() != nil {
						return ip.String(), nil
					}
				}
			}
		}
	}

	// 3. Fallback: try standard DNS resolution
	ips, err := net.LookupIP(name)
	if err != nil {
		return "", fmt.Errorf("could not resolve %q as an IP, hostname, or interface", name)
	}

	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String(), nil
		}
	}

	return ips[0].String(), nil
}
