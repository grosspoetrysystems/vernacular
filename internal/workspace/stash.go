package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// Stash moves src out of the attached checkout containing it into the workspace's apparatus,
// or working when toWorking, on the shelf named by shelf ("" for the space itself), keeping its
// name. It is the opposite of promotion: canon to workspace, a move instead of a copy. It never
// overwrites, and refuses paths with tracked files unless tracked. It returns the resolved
// source and the destination.
func Stash(src, shelf string, toWorking, tracked bool) (from, to string, err error) {
	s, err := realPath(src)
	if err != nil {
		return "", "", err
	}
	info, err := os.Lstat(s)
	if errors.Is(err, fs.ErrNotExist) {
		return "", "", fmt.Errorf("%s does not exist", src)
	} else if err != nil {
		return "", "", err
	}
	dir := s
	if !info.IsDir() {
		dir = filepath.Dir(s)
	}
	top, err := Git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", fmt.Errorf("%s is not inside a Git work tree", src)
	}
	if top, err = realPath(top); err != nil {
		return "", "", err
	}
	if s == top {
		return "", "", fmt.Errorf("%s is the repository root; stash a path inside it", src)
	}
	id, err := readLink(top)
	if err != nil {
		return "", "", err
	}
	if id == "" {
		return "", "", fmt.Errorf("repository %s is not attached to a workspace; run vern init or vern attach there", Display(top))
	}
	ws, err := loadID(id, top)
	if err != nil {
		return "", "", err
	}
	if !tracked {
		rel, _ := filepath.Rel(top, s)
		files, err := Git(top, "--literal-pathspecs", "ls-files", "--", rel)
		if err != nil {
			return "", "", err
		}
		if files != "" {
			return "", "", fmt.Errorf("%s holds tracked files; stashing deletes them from canon at your next commit\n  rerun with --tracked if that is intended", src)
		}
	}

	space := ws.Apparatus
	if toWorking {
		space = ws.Working
	}
	if shelf != "" && !filepath.IsLocal(shelf) {
		return "", "", fmt.Errorf("shelf %q must be a relative path inside the space", shelf)
	}
	dst := filepath.Join(space, shelf, filepath.Base(s))
	if _, err := os.Lstat(dst); err == nil {
		return "", "", fmt.Errorf("%s already exists; stash never overwrites", Display(dst))
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil { //nolint:gosec // ordinary shelf directories
		return "", "", err
	}
	// ponytail: rename only; a repository on another filesystem than ~/.vern gets a pointer to mv,
	// add copy-then-remove if that turns out to be common.
	if err := os.Rename(s, dst); err != nil {
		if errors.Is(err, syscall.EXDEV) {
			return "", "", fmt.Errorf("%s and the workspace are on different filesystems; move it with mv", src)
		}
		return "", "", err
	}
	return s, dst, nil
}
