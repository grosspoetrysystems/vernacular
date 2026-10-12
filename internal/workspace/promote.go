package workspace

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Entry is one filesystem object crossing into canon.
type Entry struct {
	Rel  string // relative to Plan.Src; "." when Src is a single file
	Dir  bool
	Mode fs.FileMode
	Size int64
}

// Finding is a scan hit that blocks promotion.
type Finding struct {
	Rel  string
	Line int
	Kind string
}

// Plan describes a promotion from apparatus or working into canon.
type Plan struct {
	Workspace *Workspace
	Src, Dst  string
	Entries   []Entry
	Findings  []Finding
}

// PlanPromotion validates src and dst and scans what would cross the boundary. It does not
// depend on the current directory: the workspace is the one containing src, and canon is the
// Git checkout containing dst, which must be linked to that workspace. src must be inside
// apparatus or working; dst must not exist. A dst that names an existing directory, or ends in
// a separator, receives src's base name.
func PlanPromotion(src, dst string) (*Plan, error) {
	s, err := realPath(src)
	if err != nil {
		return nil, err
	}
	ws, err := Find(filepath.Dir(s))
	if err != nil {
		return nil, err
	}
	if s == ws.Apparatus || s == ws.Working || !within(ws.Apparatus, s) && !within(ws.Working, s) {
		return nil, fmt.Errorf("%s is not inside apparatus (%s) or working (%s)", src, Display(ws.Apparatus), Display(ws.Working))
	}
	info, err := os.Lstat(s)
	if err != nil {
		return nil, err
	}

	d, err := realPath(dst)
	if err != nil {
		return nil, err
	}
	if di, err := os.Stat(d); strings.HasSuffix(dst, string(filepath.Separator)) || strings.HasSuffix(dst, "/") || err == nil && di.IsDir() {
		d = filepath.Join(d, filepath.Base(s))
	}
	if ws.Canon, err = canonFor(ws, d); err != nil {
		return nil, fmt.Errorf("%s: %w", dst, err)
	}
	if d == ws.Canon {
		return nil, fmt.Errorf("%s is the repository root, not a path inside it", dst)
	}
	if _, err := os.Lstat(d); err == nil {
		return nil, fmt.Errorf("%s already exists; promotion never overwrites canon", Display(d))
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	p := &Plan{Workspace: ws, Src: s, Dst: d}
	if !info.IsDir() {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file", src)
		}
		p.Entries = []Entry{{Rel: ".", Mode: info.Mode().Perm(), Size: info.Size()}}
	} else {
		err = filepath.WalkDir(s, func(path string, de fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			fi, err := de.Info()
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(s, path)
			switch {
			case de.IsDir():
				p.Entries = append(p.Entries, Entry{Rel: rel, Dir: true, Mode: fi.Mode().Perm()})
			case fi.Mode().IsRegular():
				p.Entries = append(p.Entries, Entry{Rel: rel, Mode: fi.Mode().Perm(), Size: fi.Size()})
			default:
				return fmt.Errorf("%s is not a regular file or directory; promote only ordinary files", path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// A file canon ignores would look promoted but never become canon.
	var dsts []string
	for _, e := range p.Entries {
		if !e.Dir {
			dsts = append(dsts, filepath.Join(d, e.Rel))
		}
	}
	if len(dsts) > 0 {
		ignored, err := Git(ws.Canon, append([]string{"check-ignore", "--"}, dsts...)...)
		var ee *exec.ExitError
		switch {
		case err == nil:
			return nil, fmt.Errorf("canon's ignore rules exclude, so these would never become canon:\n  %s",
				strings.ReplaceAll(ignored, "\n", "\n  "))
		case !errors.As(err, &ee) || ee.ExitCode() != 1:
			return nil, err
		}
	}

	home, _ := os.UserHomeDir()
	for _, e := range p.Entries {
		if e.Dir {
			continue
		}
		f, err := scanFile(filepath.Join(s, e.Rel), home)
		if err != nil {
			return nil, err
		}
		for i := range f {
			f[i].Rel = e.Rel
		}
		p.Findings = append(p.Findings, f...)
	}
	return p, nil
}

// Files returns the number of regular files in the plan.
func (p *Plan) Files() int {
	n := 0
	for _, e := range p.Entries {
		if !e.Dir {
			n++
		}
	}
	return n
}

// Apply copies the plan into canon. It refuses findings unless allowFindings and never overwrites.
// Staging and committing are left to Git and the human.
func (p *Plan) Apply(allowFindings bool) error {
	if len(p.Findings) > 0 && !allowFindings {
		return errors.New("promotion blocked by scan findings")
	}
	if err := os.MkdirAll(filepath.Dir(p.Dst), 0o755); err != nil { //nolint:gosec // ordinary directories
		return err
	}
	for _, e := range p.Entries {
		src, dst := filepath.Join(p.Src, e.Rel), filepath.Join(p.Dst, e.Rel)
		if e.Dir {
			if err := os.Mkdir(dst, e.Mode); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(src, dst, e.Mode); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src) //nolint:gosec // src was validated by PlanPromotion
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode) //nolint:gosec // dst was validated by PlanPromotion
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// ponytail: a few well-known secret shapes plus the local home path; not a security boundary.
// Swap for gitleaks if real leaks get past it.
var secretPatterns = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"private key", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
	{"AWS access key", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"GitHub token", regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{36,}`)},
	{"API key", regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}`)},
}

func scanFile(path, home string) ([]Finding, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path was validated by PlanPromotion
	if err != nil {
		return nil, err
	}
	if bytes.IndexByte(data[:min(len(data), 8000)], 0) >= 0 {
		return nil, nil // binary
	}
	var out []Finding
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(nil, len(data)+1)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if home != "" && home != "/" && strings.Contains(line, home) {
			out = append(out, Finding{Line: n, Kind: "local path"})
		}
		for _, sp := range secretPatterns {
			if sp.re.MatchString(line) {
				out = append(out, Finding{Line: n, Kind: sp.kind})
			}
		}
	}
	return out, sc.Err()
}

// canonFor returns the Git checkout containing d after checking that it belongs to ws: either ws
// names it as canon, or the checkout is attached to ws's ID. Paths inside .git
// and unlinked or nested repositories are refused.
func canonFor(ws *Workspace, d string) (string, error) {
	dir := d
	for {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	top, err := Git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not inside a Git work tree: %w", err)
	}
	if top, err = realPath(top); err != nil {
		return "", err
	}
	if top == ws.Canon {
		return top, nil
	}
	id, err := readLink(top)
	if err != nil {
		return "", err
	}
	if id == "" || id != ws.ID() {
		return "", fmt.Errorf("repository %s is not attached to workspace %s; run vern attach there", Display(top), Display(ws.Root))
	}
	return top, nil
}
