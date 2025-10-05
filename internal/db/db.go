package db

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

type CacheEntry struct {
	CreatedAt time.Time `json:"createdAt"`
	Ttl int `json:"ttl"`
	Data []byte `json:"data"`
}

type Cache struct {
	// absolute path of cache in the filesystem
	Path string
}

// Setup the memo datastore
func (c *Cache) Setup() error {
	exists, err := fileExists(c.Path)
	if err != nil {
		return fmt.Errorf("error checking if cache dir exists: %w", err)
	}

	if !exists {
		slog.Debug("Creating default data directory ~/.memo")
		err = os.Mkdir(c.Path, os.FileMode(int(0700)))
		if err != nil {
			return fmt.Errorf("error creating memo directory '%s': %w", c.Path, err)
		}
	}

	return nil
}

// Get a stored value, assigned to the supplied key
// Returns (nil, nil) if no value stored for supplied key
func (c *Cache) Get(key string) (*CacheEntry, error) {
	filePath := fmt.Sprintf("%s/%s", c.Path, key)

	// read cache if exists
	exists, err := fileExists(filePath)
	if err != nil {
		return nil, fmt.Errorf("error checking if cache file exists: %w", err)
	}
	if exists {
		slog.Debug("Cache hit", "key", key)
		existingFile, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error opening cache file: %w", err)
		}
		var e CacheEntry
		err = json.Unmarshal(existingFile, &e)
		if err != nil {
			return nil, err
		}

		return &e, nil
	}

	slog.Debug("Cache miss", "key", key)
	return nil, nil
}

// Store data for the given key
func (c *Cache) Store(key string, data []byte) error {
	filePath := fmt.Sprintf("%s/%s", c.Path, key)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file to store cached value: %w", err)
	}
	defer file.Close()

	entry := &CacheEntry{
		CreatedAt: time.Now(),
		Ttl: 3600,
		Data: data,
	}
	marshalled, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("error marshalling json: %w", err)
	}
	_, err = file.Write(marshalled)
	if err != nil {
		return fmt.Errorf("error writing data to file cache: %w", err)
	}
	slog.Debug("Wrote new cache entry", "key", key)
	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
