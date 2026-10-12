package workspace

import (
	"errors"
	"io/fs"
	"path/filepath"
)

// Count is the number of files and directories under a space, excluding the space root and .git.
type Count struct {
	Files, Dirs int
	Missing     bool
}

// CountTree counts entries under dir without following symlinks.
func CountTree(dir string) (Count, error) {
	var c Count
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch {
		case p == dir:
		case d.IsDir() && d.Name() == ".git":
			return filepath.SkipDir
		case d.IsDir():
			c.Dirs++
		default:
			c.Files++
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return Count{Missing: true}, nil
	}
	return c, err
}
