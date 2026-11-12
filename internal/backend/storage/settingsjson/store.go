package settingsjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	settingsdomain "pause/internal/backend/domain/settings"
)

type Store struct {
	path     string
	mu       sync.RWMutex
	settings settingsdomain.Settings
	created  bool
}

func OpenStore(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("config path is required")
	}

	store := &Store{path: path, settings: settingsdomain.DefaultSettings()}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) WasCreated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.created
}

func (s *Store) Get() settingsdomain.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *Store) Update(patch settingsdomain.SettingsPatch) (settingsdomain.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.settings.ApplyPatch(patch)
	if err := s.saveLocked(next); err != nil {
		return settingsdomain.Settings{}, err
	}
	s.settings = next
	return next, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
				return err
			}
			s.settings = settingsdomain.DefaultSettings()
			s.created = true
			return s.saveLocked(s.settings)
		}
		return err
	}

	settings := settingsdomain.DefaultSettings()
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			if backupErr := s.backupCorruptedConfigLocked(data); backupErr != nil {
				return errors.Join(err, backupErr)
			}
			s.settings = settingsdomain.DefaultSettings()
			return s.saveLocked(s.settings)
		}
	}

	s.settings = settings.Normalize()
	return nil
}

func (s *Store) backupCorruptedConfigLocked(data []byte) error {
	dir := filepath.Dir(s.path)
	name := filepath.Base(s.path)
	stamp := time.Now().UTC().Format("20060102-150405")
	backupPath := filepath.Join(dir, name+".corrupt."+stamp+".bak")
	return os.WriteFile(backupPath, data, 0o644)
}

func (s *Store) saveLocked(settings settingsdomain.Settings) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
