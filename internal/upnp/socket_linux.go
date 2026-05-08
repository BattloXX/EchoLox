//go:build linux

package upnp

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

// listenMulticast binds to UDP 0.0.0.0:1900 with SO_REUSEADDR so EchoLox can
// coexist with the LoxBerry system ssdpd that already holds port 1900.
// SO_REUSEADDR (not SO_REUSEPORT) is used because SO_REUSEADDR delivers a copy
// of each multicast packet to every joined socket, whereas SO_REUSEPORT
// load-balances to exactly one socket — which would mean EchoLox never sees
// M-SEARCH packets when LoxBerry's ssdpd is also running.
// SO_BROADCAST + IP_MULTICAST_IF mirror HA's emulated_hue socket setup.
func listenMulticast(localIP net.IP) (*net.UDPConn, error) {
	lc := net.ListenConfig{
		Control: func(_, _ string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
			})
		},
	}
	pc, err := lc.ListenPacket(context.Background(), "udp4", "0.0.0.0:1900")
	if err != nil {
		return nil, err
	}
	conn := pc.(*net.UDPConn)
	raw, err := conn.SyscallConn()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("syscall conn: %w", err)
	}
	lip := localIP.To4()
	if lip == nil {
		lip = net.IPv4zero.To4()
	}
	mreq := syscall.IPMreq{
		Multiaddr: [4]byte{239, 255, 255, 250},
		Interface: [4]byte{lip[0], lip[1], lip[2], lip[3]},
	}
	var setErr error
	raw.Control(func(fd uintptr) {
		// IP_MULTICAST_IF: outbound multicast on the correct interface (HA pattern)
		_, _, errno := syscall.Syscall6(
			syscall.SYS_SETSOCKOPT,
			fd,
			syscall.IPPROTO_IP,
			syscall.IP_MULTICAST_IF,
			uintptr(unsafe.Pointer(&mreq.Interface)),
			4,
			0,
		)
		if errno != 0 {
			setErr = fmt.Errorf("IP_MULTICAST_IF: %w", errno)
			return
		}
		// IP_ADD_MEMBERSHIP: receive multicast on this interface
		_, _, errno = syscall.Syscall6(
			syscall.SYS_SETSOCKOPT,
			fd,
			syscall.IPPROTO_IP,
			syscall.IP_ADD_MEMBERSHIP,
			uintptr(unsafe.Pointer(&mreq)),
			unsafe.Sizeof(mreq),
			0,
		)
		if errno != 0 {
			setErr = fmt.Errorf("IP_ADD_MEMBERSHIP: %w", errno)
		}
	})
	if setErr != nil {
		conn.Close()
		return nil, setErr
	}
	return conn, nil
}
