//go:build linux

package upnp

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

const soReusePort = 0xf // SO_REUSEPORT (Linux, all arches)

// listenMulticast binds to UDP 0.0.0.0:1900 with SO_REUSEPORT so EchoLox can
// coexist with the LoxBerry system ssdpd that already holds port 1900.
// localIP pins the multicast group join to a specific interface — same pattern
// as HA's IP_ADD_MEMBERSHIP + host_ip_addr, which fixes missed M-SEARCH packets
// on multi-homed systems (e.g. Pi with both eth0 and wlan0 active).
func listenMulticast(localIP net.IP) (*net.UDPConn, error) {
	lc := net.ListenConfig{
		Control: func(_, _ string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, soReusePort, 1)
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
