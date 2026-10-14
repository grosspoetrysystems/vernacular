package workspace

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// stashLog is the workspace's append-only journal of stashes, oldest first, one JSON object per
// line; an undo marks its entry rather than removing it. Paths are relative to canon and to the
// workspace directory, so entries survive either one moving.
const stashLog = "stash.jsonl"

// StashEntry is one recorded stash.
type StashEntry struct {
	ID       string     `json:"id"`
	At       time.Time  `json:"at"`
	Path     string     `json:"path"`                // where it was, relative to canon
	To       string     `json:"to"`                  // where it went, relative to the workspace directory
	Canon    string     `json:"canon,omitempty"`     // canon's commit at the time
	UndoneAt *time.Time `json:"undone_at,omitempty"` // set when vern stash --undo moved it back
}

// Stashed is the result of a stash or its undo. Warn is set when the move succeeded but
// recording it (in the stash log or apparatus's history) did not; the material is in place
// either way.
type Stashed struct {
	ID, From, To string
	Warn         error
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
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	inWs, _ := filepath.Rel(ws.Root, dst)
	e := StashEntry{ID: "st_" + hex.EncodeToString(b), At: time.Now().UTC().Truncate(time.Second),
		Path: filepath.ToSlash(rel), To: filepath.ToSlash(inWs), Canon: headShort(top)}
	out := Stashed{ID: e.ID, From: s, To: dst, Warn: appendEntry(filepath.Join(ws.Root, stashLog), e)}
	if toApparatus && ws.versioned() {
		kept, _ := filepath.Rel(ws.Apparatus, dst)
		out.Warn = errors.Join(out.Warn, ws.commit(kept, "Stash "+e.Path+" ("+e.ID+")"+ref))
	}
	return out, nil
}

// Unstash reverses a recorded stash, the one named by id or else the latest, for the workspace
// of the checkout containing dir, moving the material back to where it was in canon. It never
// overwrites.
func Unstash(dir, id string) (Stashed, error) {
	ws, err := Find(dir)
	if err != nil {
		return Stashed{}, err
	}
	if ws.Canon == "" {
		return Stashed{}, errors.New("run vern stash --undo inside the repository")
	}
	logPath := filepath.Join(ws.Root, stashLog)
	entries, err := readEntries(logPath)
	if err != nil {
		return Stashed{}, err
	}
	i := len(entries) - 1
	for i >= 0 && (id == "" && entries[i].UndoneAt != nil || id != "" && entries[i].ID != id) {
		i--
	}
	switch {
	case len(entries) == 0:
		return Stashed{}, errors.New("nothing to undo: no stash recorded for this workspace")
	case i < 0 && id != "":
		return Stashed{}, fmt.Errorf("no stash %s recorded for this workspace (see %s)", id, Display(logPath))
	case i < 0:
		return Stashed{}, errors.New("nothing to undo: every recorded stash is already undone")
	case entries[i].UndoneAt != nil:
		return Stashed{}, fmt.Errorf("stash %s was already undone at %s", id, entries[i].UndoneAt.Format(time.RFC3339))
	}
	e := entries[i]
	from, to := filepath.Join(ws.Root, filepath.FromSlash(e.To)), filepath.Join(ws.Canon, filepath.FromSlash(e.Path))
	if _, err := os.Lstat(from); err != nil {
		return Stashed{}, fmt.Errorf("%s is no longer in the workspace; nothing undone", Display(from))
	}
	if _, err := os.Lstat(to); err == nil {
		return Stashed{}, fmt.Errorf("%s exists again; undo never overwrites", Display(to))
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil { //nolint:gosec // recreating the original parent
		return Stashed{}, err
	}
	if err := os.Rename(from, to); err != nil {
		return Stashed{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	entries[i].UndoneAt = &now
	out := Stashed{ID: e.ID, From: from, To: to, Warn: writeEntries(logPath, entries)}
	if within(ws.Apparatus, from) && ws.versioned() {
		kept, _ := filepath.Rel(ws.Apparatus, from)
		out.Warn = errors.Join(out.Warn, ws.commit(kept, "Unstash "+e.Path+" ("+e.ID+")"+canonRef(ws.Canon)))
	}
	return out, nil
}

// headShort is canon's current commit, or "" before the first one.
func headShort(top string) string {
	sha, _ := Git(top, "rev-parse", "--short", "HEAD")
	return sha
}

func appendEntry(path string, e StashEntry) error {
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // workspace-owned log
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return errors.Join(err, f.Close())
}

func readEntries(path string) ([]StashEntry, error) {
	data, err := os.ReadFile(path) //nolint:gosec // workspace-owned log
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []StashEntry
	for n, sc := 1, bufio.NewScanner(bytes.NewReader(data)); sc.Scan(); n++ {
		if len(bytes.TrimSpace(sc.Bytes())) == 0 {
			continue
		}
		var e StashEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", Display(path), n, err)
		}
		out = append(out, e)
	}
	return out, nil
}

func writeEntries(path string, entries []StashEntry) error {
	var buf bytes.Buffer
	for _, e := range entries {
		line, err := json.Marshal(e)
		if err != nil {
			return err
		}
		buf.Write(append(line, '\n'))
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}
