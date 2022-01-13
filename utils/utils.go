package utils

import (
	"os"
	"strings"
	"time"

	"mvdan.cc/sh/v3/shell"
)

const (
	TimeFormat = "2006-01-02T15:04:05.000Z"
	// Used to indicate a file has never been synced.
	neverSyncedStr = "2000-01-01T01:01:01.000Z"
)

func RealPath(path string) (string, error) {
	escapedPath := strings.ReplaceAll(path, "'", "\\'")
	out, err := shell.Fields(escapedPath, nil)
	if err != nil {
		return "", err
	}
	return out[0], nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Remove returns a slice with the first item that satisfies f removed. Order is retained. This can be an expensive
// operation if there are many items in slice.
func Remove(f func(item StoredFile) bool, slice []StoredFile) []StoredFile {
	for i, item := range slice {
		if f(item) {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// InSlice returns true if item is present in slice.
func InSlice(item string, slice []string) bool {
	for _, sliceItem := range slice {
		if item == sliceItem {
			return true
		}
	}
	return false
}

// Returns true if the file has been synced based on the last cloud update.
func HasBeenSynced(lastCloudUpdate time.Time) bool {
	neverSynced, _ := time.Parse(TimeFormat, neverSyncedStr)
	return lastCloudUpdate.After(neverSynced)
}

// Returns a time used to signify that a file has never been synced.
func GetNeverSynced() time.Time {
	neverSynced, _ := time.Parse(TimeFormat, neverSyncedStr)
	return neverSynced
}
