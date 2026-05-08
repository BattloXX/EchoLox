//go:build !linux

package upnp

import "net"

func listenMulticast(localIP net.IP) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		return nil, err
	}
	return net.ListenMulticastUDP("udp4", interfaceByIP(localIP), addr)
}
