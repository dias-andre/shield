// Copyright (C) 2026 André de Oliveira Dias (diaso.andre@outlook.com)

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package adapters

import (
	"bytes"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/dias-andre/shield/internal/core"
)

const (
	backupMagic    = "SDBK"
	snapshotPrefix = "shield_"
	snapshotExt    = ".shieldbk"
	timeLayout     = "20060102_150405"
)

type LocalFileBackup struct {
	localPrefix   string
	encrypter     core.EncryptorPort
	keepSnapshots uint8
}

func NewLocalFileBackup(local string, port core.EncryptorPort, maxSnapshots uint8) core.BackupPort {
	var backup LocalFileBackup
	backup.localPrefix = local
	backup.encrypter = port
	backup.keepSnapshots = maxSnapshots
	return &backup
}

func (l *LocalFileBackup) deriveBackupKey(masterKey []byte) ([]byte, error) {
	prk, err := hkdf.Extract(sha256.New, masterKey, nil)
	if err != nil {
		return nil, err
	}
	info := "shield_backup"
	return hkdf.Expand(sha256.New, prk, info, 32)
}

func (l *LocalFileBackup) CreateBackup(vault *core.Vault, masterKey []byte) error {
	derivedKey, err := l.deriveBackupKey(masterKey)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(vault); err != nil {
		return err
	}
	encryptedVault, err := l.encrypter.Encrypt(buf.Bytes(), derivedKey)
	if err != nil {
		return err
	}
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(time.Now().Unix()))
	backup := make([]byte, 0, len(backupMagic)+3+8+len(encryptedVault))
	backup = append(backup, []byte(backupMagic)...)
	backup = append(backup, byte(core.ShieldMajor), byte(core.ShieldMinor), byte(core.ShieldPatch))
	backup = append(backup, timestampBytes...)
	backup = append(backup, encryptedVault...)

	filename := fmt.Sprintf("snapshot_%s.shieldbk", time.Now().Format("20060102_150405"))
	path := filepath.Join(l.localPrefix, filename)
	if err := os.MkdirAll(l.localPrefix, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, backup, 0o600); err != nil {
		return err
	}
	return l.RotateBackups(int(l.keepSnapshots))
}

func (l *LocalFileBackup) ListSnapshots() ([]core.SnapshotInfo, error) {
	entries, err := os.ReadDir(l.localPrefix)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("faield to read snapshot directory: %w", err)
	}

	snapshots := make([]core.SnapshotInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, snapshotPrefix) || !strings.HasSuffix(name, snapshotExt) {
			continue
		}

		rawTime := strings.TrimSuffix(strings.TrimPrefix(name, snapshotPrefix), snapshotExt)
		parsedTime, err := time.Parse(timeLayout, rawTime)
		if err != nil {
			continue
		}

		snapshots = append(snapshots, core.SnapshotInfo{
			Id:        filepath.Join(l.localPrefix, name),
			CreatedAt: parsedTime,
		})
	}

	slices.SortFunc(snapshots, func(a, b core.SnapshotInfo) int {
		if a.CreatedAt.After(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return 1
		}
		return 0
	})
	return snapshots, nil
}

func (l *LocalFileBackup) RotateBackups(maxKeep int) error {
	snapshots, err := l.ListSnapshots()
	if err != nil {
		return err
	}
	if len(snapshots) <= maxKeep {
		return nil
	}

	for _, oldSnapshot := range snapshots[maxKeep:] {
		if err := os.Remove(oldSnapshot.Id); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to prune old snapshot %s: %w", oldSnapshot.Id, err)
		}
	}
	return nil
}
