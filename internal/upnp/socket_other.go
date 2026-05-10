//go:build !linux

package upnp

import "net"

func listenMulticast() (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		return nil, err
	}
	return net.ListenMulticastUDP("udp4", nil, addr)
}
