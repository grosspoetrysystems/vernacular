package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// InitResult reports what Init did.
type InitResult struct {
	Workspace *Workspace
	Existing  bool   // the checkout was already attached
	Saved     string // apparatus history's first save point, when Init started it
}

// Init attaches the checkout containing repoDir to a new workspace in ~/.vern/data/ID, or, when
// it is already attached, refreshes canon's last-seen path. Tracked files are never touched.
func Init(repoDir string) (*InitResult, error) {
	top, id, err := checkout(repoDir)
	if err != nil {
		return nil, err
	}
	if id != "" {
		ws, err := loadID(id, top)
		if err != nil {
			return nil, err
		}
		saved, err := ws.keepHistory()
		if err != nil {
			return nil, fmt.Errorf("apparatus history: %w", err)
		}
		return &InitResult{Workspace: ws, Existing: true, Saved: saved}, ws.seen(top)
	}

	home, err := Home()
	if err != nil {
		return nil, err
	}
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	id = "vrn_" + hex.EncodeToString(b)
	root := filepath.Join(home, "data", id)
	ws := &Workspace{Root: root, Canon: top, Apparatus: filepath.Join(root, "apparatus"), Working: filepath.Join(root, "working")}
	ws.Manifest.ID = id
	ws.Manifest.Spaces.Canon = Space{Path: top, Kind: "canonical"}
	ws.Manifest.Spaces.Apparatus = Space{Path: "./apparatus", Kind: "retained", Visibility: "private"}
	ws.Manifest.Spaces.Working = Space{Path: "./working", Kind: "ephemeral", Visibility: "private"}
	if err := ws.checkOverlap(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	for _, d := range []string{ws.Apparatus, ws.Working} {
		if err := os.MkdirAll(d, 0o755); err != nil { //nolint:gosec // ordinary directories inside a 0700 workspace
			return nil, err
		}
	}
	if err := ws.save(); err != nil {
		return nil, err
	}
	if err := ws.register(); err != nil {
		return nil, err
	}
	saved, err := ws.keepHistory()
	if err != nil {
		return nil, fmt.Errorf("apparatus history: %w", err)
	}
	if _, err := Git(top, "config", "--local", LinkKey, id); err != nil {
		return nil, err
	}
	return &InitResult{Workspace: ws, Saved: saved}, nil
}

// Attach attaches the checkout containing repoDir to an existing workspace, named by its ID or
// by its directory. Naming the directory also (re)registers it, which recovers a lost or moved
// workspace. It returns the ID the checkout was previously attached to, if any.
func Attach(repoDir, target string) (ws *Workspace, previous string, err error) {
	top, previous, err := checkout(repoDir)
	if err != nil {
		return nil, "", err
	}
	if fi, statErr := os.Stat(target); statErr == nil && fi.IsDir() {
		if ws, err = load(target, top); err != nil {
			return nil, "", err
		}
		if ws.ID() == "" {
			return nil, "", fmt.Errorf("%s has no id; only workspaces created by vern init can be attached", Display(ws.Root))
		}
		if err := ws.register(); err != nil {
			return nil, "", err
		}
	} else if ws, err = loadID(target, top); err != nil {
		return nil, "", err
	}
	if previous != ws.ID() {
		if _, err := Git(top, "config", "--local", LinkKey, ws.ID()); err != nil {
			return nil, "", err
		}
	}
	if _, err := ws.keepHistory(); err != nil {
		return nil, "", fmt.Errorf("apparatus history: %w", err)
	}
	return ws, previous, ws.seen(top)
}

// Registered is one registered workspace.
type Registered struct {
	ID, Root, Canon string
	Err             error
}

// List returns registered workspaces sorted by ID, with canon's last-seen path.
func List() ([]Registered, error) {
	home, err := Home()
	if err != nil {
		return nil, err
	}
	files, err := filepath.Glob(filepath.Join(home, "workspaces", "*"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	var out []Registered
	for _, f := range files {
		e := Registered{ID: filepath.Base(f)}
		ws, err := loadID(e.ID, "")
		if err != nil {
			e.Err = err
		} else {
			e.Root, e.Canon = ws.Root, ws.Canon
		}
		out = append(out, e)
	}
	return out, nil
}

// seen records top as canon's last-seen path when it changed.
func (ws *Workspace) seen(top string) error {
	// A hand-written relative path describes a containing layout; leave it alone.
	if p := ws.Manifest.Spaces.Canon.Path; p == top || p != "" && !filepath.IsAbs(filepath.FromSlash(p)) {
		return nil
	}
	ws.Manifest.Spaces.Canon.Path = top
	return ws.save()
}

// checkout returns the real top-level path of the checkout containing dir and its attached ID.
func checkout(dir string) (top, id string, err error) {
	if top, err = Git(dir, "rev-parse", "--show-toplevel"); err != nil {
		return "", "", fmt.Errorf("%s is not inside a Git repository (run git init first): %w", dir, err)
	}
	if top, err = realPath(top); err != nil {
		return "", "", err
	}
	if isApparatus(top) {
		return "", "", fmt.Errorf("%s is a workspace's apparatus, not a project; run vern inside the project's repository", Display(top))
	}
	id, err = readLink(top)
	return top, id, err
}

// isApparatus reports whether top is the apparatus of the workspace directory around it.
func isApparatus(top string) bool {
	ws, err := Load(filepath.Dir(top))
	return err == nil && ws.Apparatus == top
}
