//go:build linux

package upnp

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

// listenMulticast binds to UDP 0.0.0.0:1900 with SO_REUSEPORT so EchoLox can
// coexist with the LoxBerry system ssdpd that already holds port 1900.
func listenMulticast() (*net.UDPConn, error) {
	lc := net.ListenConfig{
		Control: func(_, _ string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
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
	mreq := syscall.IPMreq{
		Multiaddr: [4]byte{239, 255, 255, 250},
	}
	var joinErr syscall.Errno
	raw.Control(func(fd uintptr) {
		_, _, joinErr = syscall.Syscall6(
			syscall.SYS_SETSOCKOPT,
			fd,
			syscall.IPPROTO_IP,
			syscall.IP_ADD_MEMBERSHIP,
			uintptr(unsafe.Pointer(&mreq)),
			unsafe.Sizeof(mreq),
			0,
		)
	})
	if joinErr != 0 {
		conn.Close()
		return nil, fmt.Errorf("join multicast group: %w", joinErr)
	}
	return conn, nil
}
