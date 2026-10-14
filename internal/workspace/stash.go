package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// Stashed is the result of a stash. Commit is set when the move into apparatus succeeded but
// recording it in apparatus's history did not; the material is in place either way.
type Stashed struct {
	From, To string
	Commit   error
}

// Stash moves src out of the attached checkout containing it into the workspace's working, or
// apparatus when toApparatus, on the shelf named by shelf ("" for the space itself), keeping its
// name. It is the opposite of promotion: canon to workspace, a move instead of a copy. It never
// overwrites, and refuses paths with tracked files unless tracked. A move into apparatus is
// committed to apparatus's history, naming canon's current commit.
func Stash(src, shelf string, toApparatus, tracked bool) (Stashed, error) {
	s, err := realPath(src)
	if err != nil {
		return Stashed{}, err
	}
	info, err := os.Lstat(s)
	if errors.Is(err, fs.ErrNotExist) {
		return Stashed{}, fmt.Errorf("%s does not exist", src)
	} else if err != nil {
		return Stashed{}, err
	}
	dir := s
	if !info.IsDir() {
		dir = filepath.Dir(s)
	}
	top, err := Git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Stashed{}, fmt.Errorf("%s is not inside a Git work tree", src)
	}
	if top, err = realPath(top); err != nil {
		return Stashed{}, err
	}
	if isApparatus(top) {
		return Stashed{}, fmt.Errorf("%s is already in the workspace's apparatus", src)
	}
	if s == top {
		return Stashed{}, fmt.Errorf("%s is the repository root; stash a path inside it", src)
	}
	id, err := readLink(top)
	if err != nil {
		return Stashed{}, err
	}
	if id == "" {
		return Stashed{}, fmt.Errorf("repository %s is not attached to a workspace; run vern init or vern attach there", Display(top))
	}
	ws, err := loadID(id, top)
	if err != nil {
		return Stashed{}, err
	}
	rel, _ := filepath.Rel(top, s)
	if !tracked {
		files, err := Git(top, "--literal-pathspecs", "ls-files", "--", rel)
		if err != nil {
			return Stashed{}, err
		}
		if files != "" {
			return Stashed{}, fmt.Errorf("%s holds tracked files; stashing deletes them from canon at your next commit\n  rerun with --tracked if that is intended", src)
		}
	}

	space := ws.Working
	if toApparatus {
		space = ws.Apparatus
	}
	if shelf != "" && !filepath.IsLocal(shelf) {
		return Stashed{}, fmt.Errorf("shelf %q must be a relative path inside the space", shelf)
	}
	dst := filepath.Join(space, shelf, filepath.Base(s))
	if _, err := os.Lstat(dst); err == nil {
		return Stashed{}, fmt.Errorf("%s already exists; stash never overwrites", Display(dst))
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Stashed{}, err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil { //nolint:gosec // ordinary shelf directories
		return Stashed{}, err
	}
	ref := canonRef(top)
	// ponytail: rename only; a repository on another filesystem than ~/.vern gets a pointer to mv,
	// add copy-then-remove if that turns out to be common.
	if err := os.Rename(s, dst); err != nil {
		if errors.Is(err, syscall.EXDEV) {
			return Stashed{}, fmt.Errorf("%s and the workspace are on different filesystems; move it with mv", src)
		}
		return Stashed{}, err
	}
	out := Stashed{From: s, To: dst}
	if toApparatus && ws.versioned() {
		kept, _ := filepath.Rel(ws.Apparatus, dst)
		out.Commit = ws.commit(kept, "Stash "+filepath.ToSlash(rel)+ref)
	}
	return out, nil
}
