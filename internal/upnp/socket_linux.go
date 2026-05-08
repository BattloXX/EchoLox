//go:build linux

package upnp

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"
)

// soReusePort is the Linux socket option constant for SO_REUSEPORT.
// syscall.SO_REUSEPORT is missing from the Go standard library on ARM targets,
// so we define the raw value directly (15 on all Linux architectures).
const soReusePort = 15

// listenMulticast creates a UDP multicast socket on 239.255.255.250:1900 with
// SO_REUSEADDR + SO_REUSEPORT so EchoLox can coexist with LoxBerry's system ssdpd.
// IP_ADD_MEMBERSHIP and IP_MULTICAST_IF are set via raw Syscall6 for ARM cross-compile
// compatibility (syscall.SetsockoptIpMreq struct layout differs on ARM).
func listenMulticast(localIP net.IP) (*net.UDPConn, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return nil, fmt.Errorf("socket: %w", err)
	}

	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("SO_REUSEADDR: %w", err)
	}
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, soReusePort, 1); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("SO_REUSEPORT: %w", err)
	}

	sa := &syscall.SockaddrInet4{Port: 1900}
	if err := syscall.Bind(fd, sa); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("bind 0.0.0.0:1900: %w", err)
	}

	// IP_ADD_MEMBERSHIP: [4]byte multicast addr + [4]byte local iface addr = 8 bytes
	mreq := [8]byte{239, 255, 255, 250, 0, 0, 0, 0}
	if ip4 := localIP.To4(); ip4 != nil && !localIP.Equal(net.IPv4zero) {
		copy(mreq[4:], ip4)
	}
	if _, _, errno := syscall.Syscall6(
		syscall.SYS_SETSOCKOPT,
		uintptr(fd), syscall.IPPROTO_IP, syscall.IP_ADD_MEMBERSHIP,
		uintptr(unsafe.Pointer(&mreq[0])), 8, 0,
	); errno != 0 {
		syscall.Close(fd)
		return nil, fmt.Errorf("IP_ADD_MEMBERSHIP: %w", errno)
	}

	// IP_MULTICAST_IF: outgoing multicast interface (non-fatal if it fails)
	if ip4 := localIP.To4(); ip4 != nil && !localIP.Equal(net.IPv4zero) {
		var iface [4]byte
		copy(iface[:], ip4)
		syscall.Syscall6(
			syscall.SYS_SETSOCKOPT,
			uintptr(fd), syscall.IPPROTO_IP, syscall.IP_MULTICAST_IF,
			uintptr(unsafe.Pointer(&iface[0])), 4, 0,
		)
	}

	f := os.NewFile(uintptr(fd), "ssdp-multicast")
	pc, err := net.FilePacketConn(f)
	f.Close() // FilePacketConn duplicates the fd
	if err != nil {
		return nil, fmt.Errorf("FilePacketConn: %w", err)
	}
	conn, ok := pc.(*net.UDPConn)
	if !ok {
		pc.Close()
		return nil, fmt.Errorf("unexpected PacketConn type")
	}
	return conn, nil
}
