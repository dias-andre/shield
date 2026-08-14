//go:build linux

package main

import (
	"log/slog"
	"net"
	"os"
	"syscall"
)

func isAuthorized(conn net.Conn) bool {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		slog.Warn("rejected connection: not a Unix domain socket")
		return false
	}
	rawConn, err := unixConn.SyscallConn()
	if err != nil {
		slog.Warn("rejected connection: failed to get syscall handle", "error", err)
		return false
	}

	var ucred *syscall.Ucred
	var syscallErr error

	err = rawConn.Control(func(fd uintptr) {
		ucred, syscallErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	if err != nil {
		slog.Warn("rejected connection: failed to read peer credentials", "error", err)
		return false
	}
	if syscallErr != nil {
		slog.Warn("rejected connection: nil credentials from kernel", "error", syscallErr)
		return false
	}
	myUID := uint32(os.Getuid())
	if ucred.Uid != myUID {
		slog.Warn("rejected connection: UID mismatch", "client_uid", ucred.Uid, "daemon_uid", myUID)
		return false
	}

	return true
}
