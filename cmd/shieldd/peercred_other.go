//go:build !linux

package main

import (
	"log/slog"
	"net"
)

func isAuthorized(conn net.Conn) bool {
	slog.Warn("peer credential verification is not supported on this platform")
	return true
}
