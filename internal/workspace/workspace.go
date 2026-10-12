// Package workspace relates three ordinary filesystem locations: canon (a Git checkout),
// apparatus, and working. Everything stays on the local machine.
//
//	checkout .git/config   vernacular.workspace = vrn_xxxxxxxx     (attachment, local Git metadata)
//	~/.vern/workspaces/ID  one line: the workspace directory       (registry pointer)
//	<workspace>/vernacular.yml  id, spaces, canon's last-seen path (self-describing; default ~/.vern/data/ID)
//
// Identity is the opaque ID; paths are only current locations. Because each workspace
// directory names its own ID, a lost or stale registry is recovered with vern attach DIR.
package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ManifestName is the workspace manifest file name.
const ManifestName = "vernacular.yml"

// LinkKey is the local Git config key attaching a checkout to a workspace ID.
const LinkKey = "vernacular.workspace"

var validID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Space is one manifest entry.
type Space struct {
	Path       string `yaml:"path,omitempty"`
	Kind       string `yaml:"kind"`
	Visibility string `yaml:"visibility,omitempty"`
}

// Manifest is the on-disk vernacular.yml.
type Manifest struct {
	Version int    `yaml:"version"`
	ID      string `yaml:"id,omitempty"`
	Spaces  struct {
		Canon     Space `yaml:"canon"`
		Apparatus Space `yaml:"apparatus"`
		Working   Space `yaml:"working"`
	} `yaml:"spaces"`
}

// Workspace is a loaded manifest with every space resolved to a real absolute path.
type Workspace struct {
	Root     string // directory holding vernacular.yml
	Manifest Manifest
	// Canon is the checkout in use, else canon's last-seen path, else "".
	Canon     string
	Apparatus string
	Working   string
}

// ID is the workspace's opaque identity, or "" for a hand-written manifest.
func (ws *Workspace) ID() string { return ws.Manifest.ID }

// Home is Vernacular's local directory, ~/.vern.
func Home() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".vern"), nil
}

// Find resolves the workspace for dir. Inside an attached checkout, the attachment wins and canon
// is that checkout, so moved repositories and worktrees resolve to themselves. Otherwise Find
// walks up to the nearest vernacular.yml: the workspace directory itself, or a layout whose
// workspace contains its repository.
func Find(dir string) (*Workspace, error) {
	dir, err := realPath(dir)
	if err != nil {
		return nil, err
	}
	if top, err := Git(dir, "rev-parse", "--show-toplevel"); err == nil {
		id, err := readLink(top)
		if err != nil {
			return nil, err
		}
		if id != "" {
			if top, err = realPath(top); err != nil {
				return nil, err
			}
			return loadID(id, top)
		}
	}
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ManifestName)); err == nil {
			return Load(d)
		}
		if filepath.Dir(d) == d {
			return nil, fmt.Errorf("no workspace found for %s; run vern init or vern attach inside the repository", dir)
		}
	}
}

// loadID resolves a registered ID through its pointer and checks the manifest agrees.
func loadID(id, canon string) (*Workspace, error) {
	root, err := lookup(id)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(root, ManifestName)); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("workspace %s is registered at %s, which has no %s\n  if it moved: vern attach NEW_DIR", id, Display(root), ManifestName)
	}
	ws, err := load(root, canon)
	if err != nil {
		return nil, err
	}
	if ws.ID() != id {
		return nil, fmt.Errorf("%s holds workspace %q, not %s\n  repair: vern attach DIR_OF_%s", Display(root), ws.ID(), id, id)
	}
	return ws, nil
}

// lookup reads the registry pointer for id.
func lookup(id string) (string, error) {
	if !validID.MatchString(id) {
		return "", fmt.Errorf("invalid workspace id %q", id)
	}
	home, err := Home()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(home, "workspaces", id)) //nolint:gosec // id is validated
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("workspace %s is not registered in %s\n  if its directory still exists: vern attach DIR\n  to start fresh:                 git config --unset %s && vern init",
			id, Display(filepath.Join(home, "workspaces")), LinkKey)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// register points ~/.vern/workspaces/ID at ws.Root.
func (ws *Workspace) register() error {
	home, err := Home()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, "workspaces")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ws.ID()), []byte(ws.Root+"\n"), 0o600)
}

// Load reads root/vernacular.yml and resolves its spaces.
func Load(root string) (*Workspace, error) { return load(root, "") }

// load resolves the manifest at root. A non-empty canon overrides spaces.canon.path.
func load(root, canon string) (*Workspace, error) {
	root, err := realPath(root)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, ManifestName)
	data, err := os.ReadFile(path) //nolint:gosec // path is the workspace manifest by construction
	if err != nil {
		return nil, err
	}
	ws := &Workspace{Root: root}
	if err := yaml.Unmarshal(data, &ws.Manifest); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if ws.Manifest.Version != 0 {
		return nil, fmt.Errorf("%s: unsupported version %d", path, ws.Manifest.Version)
	}
	if ws.ID() != "" && !validID.MatchString(ws.ID()) {
		return nil, fmt.Errorf("%s: invalid id %q", path, ws.ID())
	}
	if p := ws.Manifest.Spaces.Canon.Path; p != "" {
		if ws.Canon, err = resolve(root, p); err != nil {
			return nil, err
		}
	}
	for _, s := range []struct {
		name string
		rel  string
		dst  *string
	}{
		{"apparatus", ws.Manifest.Spaces.Apparatus.Path, &ws.Apparatus},
		{"working", ws.Manifest.Spaces.Working.Path, &ws.Working},
	} {
		if s.rel == "" {
			return nil, fmt.Errorf("%s: spaces.%s.path is required", path, s.name)
		}
		if *s.dst, err = resolve(root, s.rel); err != nil {
			return nil, err
		}
	}
	if canon != "" {
		ws.Canon = canon
	}
	return ws, ws.checkOverlap()
}

// checkOverlap enforces the invariant the model rests on: private spaces live outside canon.
func (ws *Workspace) checkOverlap() error {
	for _, p := range []string{ws.Apparatus, ws.Working} {
		if ws.Canon != "" && (within(ws.Canon, p) || within(p, ws.Canon)) {
			return fmt.Errorf("%s: %s overlaps the canonical repository %s", filepath.Join(ws.Root, ManifestName), p, ws.Canon)
		}
	}
	return nil
}

func (ws *Workspace) save() error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(ws.Manifest); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(ws.Root, ManifestName), buf.Bytes(), 0o600)
}

// Abbrev renders p as ~/relative inside home, else absolute. Use it where the reader may not
// share the current directory: summaries and listings.
func Abbrev(p string) string {
	if h, err := os.UserHomeDir(); err == nil {
		if h, err = realPath(h); err == nil && within(h, p) {
			r, _ := filepath.Rel(h, p)
			if r == "." {
				return "~"
			}
			return "~" + string(filepath.Separator) + r
		}
	}
	return p
}

// Display renders p for humans: ./relative inside the current directory, else as Abbrev.
func Display(p string) string {
	rel := func(base, prefix string) (string, bool) {
		base, err := realPath(base)
		if err != nil || !within(base, p) {
			return "", false
		}
		r, _ := filepath.Rel(base, p)
		if r == "." {
			return prefix, true
		}
		return prefix + string(filepath.Separator) + r, true
	}
	if wd, err := os.Getwd(); err == nil {
		if s, ok := rel(wd, "."); ok {
			return s
		}
	}
	return Abbrev(p)
}

// readLink returns the checkout's workspace ID, or "" when unattached.
func readLink(repo string) (string, error) {
	out, err := Git(repo, "config", "--get", LinkKey)
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return "", nil
	}
	return out, err
}

func resolve(root, p string) (string, error) {
	p = filepath.FromSlash(p)
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return realPath(p)
}

// within reports whether p is base or inside base. Both must be clean absolute paths.
func within(base, p string) bool {
	r, err := filepath.Rel(base, p)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

// realPath returns an absolute path with symlinks resolved in its longest existing prefix,
// so containment checks hold for paths that do not exist yet.
func realPath(p string) (string, error) {
	p, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	var rest []string
	for {
		r, err := filepath.EvalSymlinks(p)
		if err == nil {
			return filepath.Join(append([]string{r}, rest...)...), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(p)
		if parent == p {
			return "", err
		}
		rest = append([]string{filepath.Base(p)}, rest...)
		p = parent
	}
}

// Git runs the installed git executable and returns trimmed stdout.
func Git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...) //nolint:gosec // fixed git binary; args are vern-built
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(stderr.String()), err)
	}
	return strings.TrimSpace(string(out)), nil
}
