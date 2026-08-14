// Package api defines the RPC request and reply types used by the daemon and the CLI.
package api

import "github.com/dias-andre/shield/internal/core"

type ServerEntry struct {
	Name string
	User string
	Host string
}

type CreateSSHEntryRequest struct {
	Name        string
	User        string
	Host        string
	AuthType    core.AuthMethod
	KeyLocation string
}

type RemoveSSHEntryRequest struct {
	Name string
}

type GetServerEntryRequest struct {
	Name string
}

type RemoveSSHEntryReply struct {
	Success  bool
	ErrorMsg string
}

type EmptyRequest struct{}

type FetchEntriesReply struct {
	Entries []ServerEntry
}

type CreateSSHEntryReply struct {
	Success   bool
	ErrorCode int
	ErrorMsg  string
}

type GetServerEntryReply struct {
	Entry   ServerEntry
	Success bool
}
