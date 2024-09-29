package db

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

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
		err = os.Mkdir(c.Path, os.FileMode(int(0700)))
		if err != nil {
			return fmt.Errorf("error creating memo directory '%s': %w", c.Path, err)
		}
	}

	return nil
}

// Get a stored value, assigned to the supplied key
// Returns (nil, nil) if no value stored for supplied key
func (c *Cache) Get(key string) (*os.File, error) {
	filePath := fmt.Sprintf("%s/%s", c.Path, key)

	// read cache if exists
	exists, err := fileExists(filePath)
	if err != nil {
		return nil, fmt.Errorf("error checking if cache file exists: %w", err)
	}
	if exists {
		existingFile, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("error opening cache file: %w", err)
		}
		return existingFile, nil
	}

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

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("error writing data to file cache: %w", err)
	}
	return nil
}

type header struct {
    contentLength int64
    updatedAt int64
}

func (h *header) toString() string {
    return fmt.Sprintf("memo,%d,%d\n", h.contentLength, h.updatedAt)
}

func parseHeader(data []byte) (*header, error) {
    headerHeader := string(data[:4])
    if headerHeader != "memo" {
        return nil, errors.New("Data does not begin with a memo header")
    }

    i := 0
    for {
        r,_ := utf8.DecodeRune([]byte{data[i]})
        if r == utf8.RuneError {
            return nil, errors.New("Error parsing memo header, encountered error Rune")
        }
        if r == '\n' {
            break
        }
    }
    if i == 0 {
        panic("i was still 0")
    }

    headerString := string(data[:i])
    vals := strings.Split(headerString, ",")

    // see header::toString for order
    contentLength, err := strconv.ParseInt(vals[1])
    if err != nil {
        return nil, fmt.Errorf("Error parsing contentLength integer header: %w", err)
    }
    updatedAt, err := strconv.ParseInt(vals[2])
    if err != nil {
        return nil, fmt.Errorf("Error parsing updatedAt integer header: %w", err)
    }
    return &header{contentLength: contentLength, updatedAt: updatedAt}, nil



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
