package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Burning is what burning the workspace of a checkout removes: the workspace directory with
// everything in it (apparatus history included), its registry pointer, and the checkout's link.
type Burning struct {
	ID, Root, Pointer, Canon string
	Apparatus, Working       Tree     // what goes, for spaces inside Root
	Unsaved                  int      // apparatus changes not in its history
	Kept                     []string // spaces outside Root; the human chose them, so they stay
	Gone                     bool     // Root no longer holds the workspace; only leftovers remain
}

// PlanBurn reports what burning the workspace attached to the checkout containing dir would
// remove, without removing anything. It refuses a workspace directory that contains the
// checkout, or that holds a different workspace than the link names.
func PlanBurn(dir string) (*Burning, error) {
	top, id, err := checkout(dir)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("%s is not attached to a workspace; nothing to burn", Display(top))
	}
	home, err := Home()
	if err != nil {
		return nil, err
	}
	b := &Burning{ID: id, Canon: top, Pointer: filepath.Join(home, "workspaces", id)}
	if b.Root, err = lookup(id); err != nil {
		if _, serr := os.Stat(b.Pointer); errors.Is(serr, fs.ErrNotExist) {
			b.Gone = true // nothing registered and nothing in the default place: only the link is left
			return b, nil
		}
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(b.Root, ManifestName)); errors.Is(err, fs.ErrNotExist) {
		b.Gone = true // never delete a directory that no longer holds the workspace
		return b, nil
	}
	ws, err := loadID(id, top)
	if err != nil {
		return nil, err
	}
	if b.Root, err = realPath(ws.Root); err != nil {
		return nil, err
	}
	if within(b.Root, top) {
		return nil, fmt.Errorf("%s contains this repository; burning it would delete the project too", Display(b.Root))
	}
	for _, s := range []struct {
		path string
		tree *Tree
	}{{ws.Apparatus, &b.Apparatus}, {ws.Working, &b.Working}} {
		if !within(b.Root, s.path) {
			b.Kept = append(b.Kept, s.path)
			continue
		}
		if *s.tree, err = ScanTree(s.path, 0); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	b.Unsaved, _ = ws.Unrecorded()
	return b, nil
}

// Apply removes the workspace directory, its registry pointer, and the checkout's link, in that
// order, so an interrupted burn can simply be run again.
func (b *Burning) Apply() error {
	if !b.Gone {
		if err := os.RemoveAll(b.Root); err != nil {
			return err
		}
	}
	if err := os.Remove(b.Pointer); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	_, err := Git(b.Canon, "config", "--local", "--unset", LinkKey)
	return err
}
