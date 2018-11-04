package bindmnt

import (
	"fmt"
	"path/filepath"
)

func Resolve(path string) (string, error) {
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks in %v: %v", path, err)
	}

	return resolve(path)
}
