package workspace

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Tree summarizes a space: counts of files and directories (excluding the space root and .git)
// and its most recently modified files, newest first.
type Tree struct {
	Files, Dirs int
	Recent      []Item
	Missing     bool
}

// Item is a file in a space, by path relative to the space root.
type Item struct {
	Path string
	Mod  time.Time
}

// ScanTree summarizes dir without following symlinks, keeping at most n recent files. Dotfiles
// are counted but never listed.
func ScanTree(dir string, n int) (Tree, error) {
	var t Tree
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch {
		case p == dir:
		case d.IsDir() && d.Name() == ".git":
			return filepath.SkipDir
		case d.IsDir():
			t.Dirs++
		default:
			t.Files++
			if strings.HasPrefix(d.Name(), ".") {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(dir, p)
			t.Recent = append(t.Recent, Item{rel, info.ModTime()})
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return Tree{Missing: true}, nil
	}
	sort.Slice(t.Recent, func(i, j int) bool { return t.Recent[i].Mod.After(t.Recent[j].Mod) })
	if len(t.Recent) > n {
		t.Recent = t.Recent[:n]
	}
	return t, err
}
