package upnp

import "net"

func listenMulticast(localIP net.IP) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		return nil, err
	}
	return net.ListenMulticastUDP("udp4", interfaceByIP(localIP), addr)
}

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
