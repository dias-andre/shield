package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dias-andre/shield/internal/core"
)

type InitializationStore struct {
	mu    sync.RWMutex
	path  string
	state core.InitializationRecord
}

func NewInitializationStore(vaultDir string) *InitializationStore {
	return &InitializationStore{
		path: filepath.Join(vaultDir, ".init-state.json"),
	}
}

func (is *InitializationStore) Load() (*core.InitializationRecord, error) {
	is.mu.RLock()
	defer is.mu.RUnlock()

	data, err := os.ReadFile(is.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &core.InitializationRecord{
				State:     core.InitNotStarted,
				Timestamp: time.Now(),
			}, nil
		}
		return nil, err
	}

	var record core.InitializationRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}

	return &record, nil
}

func (is *InitializationStore) Save(record *core.InitializationRecord) error {
	is.mu.Lock()
	defer is.mu.Unlock()

	record.Timestamp = time.Now()
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(is.path, data, 0o600)
}
