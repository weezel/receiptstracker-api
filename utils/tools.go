package utils

import (
	"os"
	"path/filepath"
	"receiptstracker-api/external"
	"strings"
)

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func DeleteFromSlice(a []string, i int) []string {
	return append(a[:i], a[i+1:]...)
}

func IsAllowedFileExt(fname string) bool {
	if strings.Index(fname, ".") == -1 {
		return false
	}

	fileExt := strings.Trim(
		strings.ToLower(filepath.Ext(fname)),
		".",
	)
	for _, ext := range external.AllowedExtensions {
		if ext == fileExt {
			return true
		}
	}
	return false
}
