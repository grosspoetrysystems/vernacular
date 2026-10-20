package workspace

import (
	"fmt"
	"path/filepath"
	"strings"
)

// versioned reports whether vern keeps apparatus's history: apparatus is in its default place
// inside the workspace, not a path the human chose (a shared clone, a notes folder), and is the
// top of its own Git repository.
func (ws *Workspace) versioned() bool {
	if ws.Apparatus != filepath.Join(ws.Root, "apparatus") {
		return false
	}
	top, err := Git(ws.Apparatus, "rev-parse", "--show-toplevel")
	if err != nil {
		return false
	}
	top, err = realPath(top)
	return err == nil && top == ws.Apparatus
}

// keepHistory makes a default-placed apparatus its own Git repository, so stashed and compacted
// material can always be recovered, and commits whatever is already there. It returns that save
// point ("" when apparatus already had history or nothing to save). Apparatus paths the human
// chose are theirs to version.
func (ws *Workspace) keepHistory() (string, error) {
	if ws.Apparatus != filepath.Join(ws.Root, "apparatus") || ws.versioned() {
		return "", nil
	}
	if _, err := Git(ws.Apparatus, "init", "-q"); err != nil {
		return "", err
	}
	return ws.commit(".", "Start apparatus history")
}

// commit saves rel (relative to apparatus) in apparatus's history and returns the save point's
// short commit, or "" when there was nothing to save. vern's own commits never sign, so a signing
// setup cannot block or hang them, and fall back to a vern identity when the human has none; the
// repository's config is left as it is.
func (ws *Workspace) commit(rel, message string) (string, error) {
	if _, err := Git(ws.Apparatus, "--literal-pathspecs", "add", "-A", "--", rel); err != nil {
		return "", err
	}
	if _, err := Git(ws.Apparatus, "--literal-pathspecs", "diff", "--cached", "--quiet", "--", rel); err == nil {
		return "", nil
	}
	args := []string{"-c", "commit.gpgsign=false"}
	if email, _ := Git(ws.Apparatus, "config", "user.email"); email == "" {
		args = append(args, "-c", "user.name=vern", "-c", "user.email=vern@localhost")
	}
	if _, err := Git(ws.Apparatus, append(args, "--literal-pathspecs", "commit", "-q", "-m", message, "--", rel)...); err != nil {
		return "", err
	}
	return Git(ws.Apparatus, "rev-parse", "--short", "HEAD")
}

// Unrecorded counts apparatus paths with changes not yet in its history; ok is false when vern
// does not keep apparatus's history.
func (ws *Workspace) Unrecorded() (n int, ok bool) {
	if !ws.versioned() {
		return 0, false
	}
	out, err := Git(ws.Apparatus, "status", "--porcelain")
	if err != nil || out == "" {
		return 0, err == nil
	}
	return strings.Count(out, "\n") + 1, true
}

// canonRef names canon's current commit for a commit trailer, or "" before its first commit.
func canonRef(top string) string {
	sha, err := Git(top, "rev-parse", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return fmt.Sprintf("\n\nCanon: %s %s", filepath.Base(top), strings.TrimSpace(sha))
}
