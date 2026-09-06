package dpi

import (
	"net"
)

// SetSocketTTL sets the IP Time-To-Live (Hop Limit) on the underlying TCP socket.
// This allows low-level packet crafting where fake decoy packets are injected with
// a low TTL (e.g. 3-5 hops) so they reach and desynchronize the ISP DPI middlebox
// but expire before reaching the destination server.
func SetSocketTTL(conn net.Conn, ttl int) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil
	}

	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}

	var sockErr error
	err = rawConn.Control(func(fd uintptr) {
		sockErr = setTTLNative(fd, ttl)
	})
	if err != nil {
		return err
	}
	return sockErr
}

// SetSocketMSS sets the Maximum Segment Size (TCP_MAXSEG) on the underlying socket.
// This forces hardware/kernel-level fragmentation of outgoing TCP segments into small
// chunks (e.g. 64-128 bytes) without userland slicing overhead.
func SetSocketMSS(conn net.Conn, mss int) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return net.UnknownNetworkError("not a TCPConn")
	}

	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}

	var sockErr error
	err = rawConn.Control(func(fd uintptr) {
		sockErr = setMSSNative(fd, mss)
	})
	if err != nil {
		return err
	}
	return sockErr
}
