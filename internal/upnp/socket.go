package upnp

import "net"

// interfaceByIP returns the network interface that has localIP assigned,
// or nil (all interfaces) if not found.
func interfaceByIP(ip net.IP) *net.Interface {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var a net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				a = v.IP
			case *net.IPAddr:
				a = v.IP
			}
			if a != nil && a.Equal(ip) {
				return &iface
			}
		}
	}
	return nil
}
