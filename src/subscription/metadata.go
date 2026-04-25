package subscription

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"routex/utils/intID"
)

type Metadata struct {
	LastUpdated  time.Time `json:"lastUpdated"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"lastModified,omitempty"`
	RuleCount    int       `json:"ruleCount"`
	LastError    string    `json:"lastError,omitempty"`
	LastErrorAt  time.Time `json:"lastErrorAt,omitempty"`
}

func cacheDir(stateDir string) string {
	return filepath.Join(stateDir, "subscriptions")
}

func listCachePath(stateDir string, groupID intID.ID) string {
	return filepath.Join(cacheDir(stateDir), groupID.String()+".list")
}

func metaCachePath(stateDir string, groupID intID.ID) string {
	return filepath.Join(cacheDir(stateDir), groupID.String()+".meta.json")
}

func ensureCacheDir(stateDir string) error {
	return os.MkdirAll(cacheDir(stateDir), 0755)
}

func loadMetadata(stateDir string, groupID intID.ID) (*Metadata, error) {
	data, err := os.ReadFile(metaCachePath(stateDir, groupID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Metadata{}, nil
		}
		return nil, err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return &Metadata{}, nil
	}
	return &meta, nil
}

func saveMetadata(stateDir string, groupID intID.ID, meta *Metadata) error {
	if err := ensureCacheDir(stateDir); err != nil {
		return err
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(metaCachePath(stateDir, groupID), data, 0600)
}

func saveCachedList(stateDir string, groupID intID.ID, body []byte) error {
	if err := ensureCacheDir(stateDir); err != nil {
		return err
	}
	tmp := listCachePath(stateDir, groupID) + ".tmp"
	if err := os.WriteFile(tmp, body, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, listCachePath(stateDir, groupID))
}

func loadCachedList(stateDir string, groupID intID.ID) ([]byte, error) {
	data, err := os.ReadFile(listCachePath(stateDir, groupID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func removeCachedFiles(stateDir string, groupID intID.ID) {
	_ = os.Remove(listCachePath(stateDir, groupID))
	_ = os.Remove(metaCachePath(stateDir, groupID))
}
