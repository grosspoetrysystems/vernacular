package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepo creates a Git repository named name in a fresh temp directory and returns its real path.
func newRepo(t *testing.T, name string, commit bool) string {
	t.Helper()
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(parent, name)
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, repo, "init", "-q")
	if commit {
		run(t, repo, "commit", "-q", "--allow-empty", "-m", "root "+name+" "+parent)
	}
	return repo
}

// isolate points HOME at a temp directory so ~/.vern is per test.
func isolate(t *testing.T) string {
	t.Helper()
	h, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", h)
	return filepath.Join(h, ".vern")
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustInit(t *testing.T, repo string) *Workspace {
	t.Helper()
	res, err := Init(repo)
	if err != nil {
		t.Fatal(err)
	}
	return res.Workspace
}

func TestInitAttachesWithoutTouchingTrackedFiles(t *testing.T) {
	home := isolate(t)
	repo := newRepo(t, "app", false)
	write(t, filepath.Join(repo, "PLAN.md"), "existing mess stays put\n")
	write(t, filepath.Join(repo, "src", "main.go"), "package main\n")

	ws := mustInit(t, repo)
	if !strings.HasPrefix(ws.ID(), "vrn_") || ws.Root != filepath.Join(home, "data", ws.ID()) {
		t.Errorf("workspace %s at %s", ws.ID(), ws.Root)
	}
	if got := run(t, repo, "config", "--get", LinkKey); got != ws.ID() {
		t.Errorf("attachment = %q, want %q", got, ws.ID())
	}
	if ptr, err := os.ReadFile(filepath.Join(home, "workspaces", ws.ID())); err != nil || strings.TrimSpace(string(ptr)) != ws.Root {
		t.Errorf("registry pointer = %q, %v", ptr, err)
	}
	if got := run(t, repo, "status", "--porcelain"); got != "?? PLAN.md\n?? src/" {
		t.Errorf("init changed canon; git status = %q", got)
	}
	found, err := Find(filepath.Join(repo, "src"))
	if err != nil || found.ID() != ws.ID() || found.Canon != repo {
		t.Fatalf("Find from inside canon = %+v, %v", found, err)
	}
	again, err := Init(repo)
	if err != nil || !again.Existing || again.Workspace.ID() != ws.ID() {
		t.Errorf("second init = %+v, %v", again, err)
	}
}

func TestAttachmentFollowsMovesAndWorktrees(t *testing.T) {
	isolate(t)
	repo := newRepo(t, "app", true)
	ws := mustInit(t, repo)

	moved := filepath.Join(filepath.Dir(repo), "renamed")
	if err := os.Rename(repo, moved); err != nil {
		t.Fatal(err)
	}
	if found, err := Find(moved); err != nil || found.Canon != moved || found.ID() != ws.ID() {
		t.Fatalf("after move Find = %+v, %v", found, err)
	}
	tree := filepath.Join(filepath.Dir(moved), "feature")
	run(t, moved, "worktree", "add", "-q", tree)
	if found, err := Find(tree); err != nil || found.Canon != tree || found.ID() != ws.ID() {
		t.Errorf("worktree Find = %+v, %v", found, err)
	}
	// Init in a moved checkout records the new last-seen path.
	if _, err := Init(moved); err != nil {
		t.Fatal(err)
	}
	if reloaded, err := Load(ws.Root); err != nil || reloaded.Canon != moved {
		t.Errorf("last seen = %+v, %v", reloaded, err)
	}
}

func TestRecoveryThroughAttachDir(t *testing.T) {
	home := isolate(t)
	repo := newRepo(t, "app", true)
	ws := mustInit(t, repo)

	// Lost registry: the data directory still describes itself.
	if err := os.RemoveAll(filepath.Join(home, "workspaces")); err != nil {
		t.Fatal(err)
	}
	if _, err := Find(repo); err == nil || !strings.Contains(err.Error(), "vern attach DIR") {
		t.Fatalf("Find with lost registry: err = %v", err)
	}
	if _, _, err := Attach(repo, ws.Root); err != nil {
		t.Fatal(err)
	}

	// Moved data directory: the stale pointer explains itself; attach DIR repoints it.
	elsewhere := filepath.Join(filepath.Dir(home), "disk2", "app-ws")
	if err := os.MkdirAll(filepath.Dir(elsewhere), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(ws.Root, elsewhere); err != nil {
		t.Fatal(err)
	}
	if _, err := Find(repo); err == nil || !strings.Contains(err.Error(), "vern attach NEW_DIR") {
		t.Fatalf("Find with moved data: err = %v", err)
	}
	got, previous, err := Attach(repo, elsewhere)
	if err != nil || previous != ws.ID() || got.ID() != ws.ID() {
		t.Fatalf("attach moved dir = %+v, %q, %v", got, previous, err)
	}
	if found, err := Find(repo); err != nil || found.Root != elsewhere {
		t.Errorf("after recovery Find = %+v, %v", found, err)
	}
}

func TestFreshCloneAttachesByID(t *testing.T) {
	isolate(t)
	repo := newRepo(t, "app", true)
	ws := mustInit(t, repo)
	clone := filepath.Join(filepath.Dir(newRepo(t, "unused", false)), "clone")
	run(t, filepath.Dir(clone), "clone", "-q", repo, clone)

	if _, err := Find(clone); err == nil {
		t.Fatal("a fresh clone must not inherit an attachment")
	}
	list, err := List()
	if err != nil || len(list) != 1 || list[0].ID != ws.ID() || list[0].Canon != repo {
		t.Fatalf("List = %+v, %v", list, err)
	}
	if _, _, err := Attach(clone, ws.ID()); err != nil {
		t.Fatal(err)
	}
	if found, err := Find(clone); err != nil || found.ID() != ws.ID() || found.Canon != clone {
		t.Errorf("Find in clone = %+v, %v", found, err)
	}
}

func TestAttachRefusals(t *testing.T) {
	isolate(t)
	repo := newRepo(t, "app", true)
	nested := t.TempDir()
	write(t, filepath.Join(nested, ManifestName), "version: 0\nspaces:\n  apparatus: {path: ./a}\n  working: {path: ./w}\n")
	cases := []struct{ name, repo, target, want string }{
		{"unknown id", repo, "vrn_00000000", "not registered"},
		{"path-like id", repo, "../../etc", "invalid workspace id"},
		{"manifest without id", repo, nested, "has no id"},
		{"not a repository", t.TempDir(), "vrn_00000000", "not inside a Git repository"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := Attach(tc.repo, tc.target); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestLoadRejectsPrivateSpaceInsideCanon(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ManifestName), `version: 0
spaces:
  canon: {path: ./project}
  apparatus: {path: ./project/apparatus}
  working: {path: ./working}
`)
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "overlaps the canonical repository") {
		t.Errorf("err = %v", err)
	}
}

func TestPromote(t *testing.T) {
	isolate(t)
	repo := newRepo(t, "app", true)
	ws := mustInit(t, repo)
	write(t, filepath.Join(ws.Apparatus, "research", "parser.md"), "# Parser\nfindings\n")
	write(t, filepath.Join(ws.Working, "bench", "a.csv"), "a,b\n")
	write(t, filepath.Join(ws.Working, "bench", "nested", "b.csv"), "c,d\n")
	write(t, filepath.Join(repo, "docs", "keep.md"), "canon\n")
	write(t, filepath.Join(repo, ".gitignore"), "generated/\n")
	unlinked := newRepo(t, "unlinked", true)

	t.Run("file into existing directory keeps its name", func(t *testing.T) {
		p, err := PlanPromotion(filepath.Join(ws.Apparatus, "research", "parser.md"), filepath.Join(repo, "docs"))
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Apply(false); err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(repo, "docs", "parser.md")); err != nil || string(got) != "# Parser\nfindings\n" {
			t.Fatalf("got %q, %v", got, err)
		}
		if _, err := os.Stat(filepath.Join(ws.Apparatus, "research", "parser.md")); err != nil {
			t.Errorf("source should remain after promotion: %v", err)
		}
	})

	t.Run("directory copies recursively to new path", func(t *testing.T) {
		p, err := PlanPromotion(filepath.Join(ws.Working, "bench"), filepath.Join(repo, "testdata", "bench"))
		if err != nil {
			t.Fatal(err)
		}
		if p.Files() != 2 {
			t.Errorf("files = %d, want 2", p.Files())
		}
		if err := p.Apply(false); err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(repo, "testdata", "bench", "nested", "b.csv")); err != nil || string(got) != "c,d\n" {
			t.Fatalf("got %q, %v", got, err)
		}
	})

	a := filepath.Join(ws.Working, "bench", "a.csv")
	refusals := []struct {
		name, src, dst, want string
	}{
		{"source in canon", filepath.Join(repo, "docs", "keep.md"), filepath.Join(repo, "x.md"), "not inside apparatus"},
		{"whole apparatus", ws.Apparatus, filepath.Join(repo, "all"), "not inside apparatus"},
		{"dest outside any repository", a, filepath.Join(t.TempDir(), "a.csv"), "not inside a Git work tree"},
		{"dest in .git", a, filepath.Join(repo, ".git", "hooks", "x"), "not inside a Git work tree"},
		{"dest in an unlinked repository", a, filepath.Join(unlinked, "a.csv"), "not attached to workspace"},
		{"overwrite", a, filepath.Join(repo, "docs", "keep.md"), "never overwrites"},
		{"ignored by canon", a, filepath.Join(repo, "generated") + "/", "would never become canon"},
	}
	for _, tc := range refusals {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PlanPromotion(tc.src, tc.dst)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestStash(t *testing.T) {
	isolate(t)
	repo := newRepo(t, "app", true)
	ws := mustInit(t, repo)
	write(t, filepath.Join(repo, ".substrate", ".gitignore"), "*\n")
	write(t, filepath.Join(repo, ".substrate", "adr", "0001.md"), "decision\n")
	write(t, filepath.Join(repo, "PLAN.md"), "plan\n")
	write(t, filepath.Join(repo, "scratch.log"), "log\n")
	write(t, filepath.Join(repo, "README.md"), "canon\n")
	write(t, filepath.Join(repo, "ab.md"), "tracked\n")
	write(t, filepath.Join(repo, "a*.md"), "untracked, glob-like name\n")
	run(t, repo, "add", "README.md", "ab.md")
	run(t, repo, "commit", "-q", "-m", "readme")
	write(t, filepath.Join(ws.Apparatus, "plans", "PLAN.md"), "already shelved\n")
	unlinked := newRepo(t, "unlinked", true)
	write(t, filepath.Join(unlinked, "notes.md"), "x\n")

	t.Run("directory moves intact onto a nested shelf", func(t *testing.T) {
		st, err := Stash(filepath.Join(repo, ".substrate", "adr"), "research/old", true, false)
		if err != nil {
			t.Fatal(err)
		}
		dst := st.To
		if want := filepath.Join(ws.Apparatus, "research", "old", "adr"); dst != want {
			t.Errorf("dst = %s, want %s", dst, want)
		}
		if got, err := os.ReadFile(filepath.Join(dst, "0001.md")); err != nil || string(got) != "decision\n" {
			t.Fatalf("got %q, %v", got, err)
		}
		if _, err := os.Stat(filepath.Join(repo, ".substrate", "adr")); !os.IsNotExist(err) {
			t.Errorf("source should be gone after stash: %v", err)
		}
	})

	t.Run("glob-like name is matched literally", func(t *testing.T) {
		// As a pathspec, "a*.md" would match the tracked ab.md and refuse an untracked file.
		if _, err := Stash(filepath.Join(repo, "a*.md"), "", false, false); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("working by default, without a shelf", func(t *testing.T) {
		if st, err := Stash(filepath.Join(repo, "scratch.log"), "", false, false); err != nil || st.To != filepath.Join(ws.Working, "scratch.log") {
			t.Fatalf("dst %s, err %v", st.To, err)
		}
	})

	t.Run("tracked only with consent", func(t *testing.T) {
		if _, err := Stash(filepath.Join(repo, "README.md"), "", false, false); err == nil || !strings.Contains(err.Error(), "tracked") {
			t.Fatalf("err = %v, want tracked refusal", err)
		}
		if _, err := Stash(filepath.Join(repo, "README.md"), "", false, true); err != nil {
			t.Fatal(err)
		}
		if st := run(t, repo, "status", "--short", "README.md"); st != "D README.md" {
			t.Errorf("git status = %q, want the deletion left for the human", st)
		}
	})

	refusals := []struct {
		name, src, shelf, want string
	}{
		{"overwrite", filepath.Join(repo, "PLAN.md"), "plans", "never overwrites"},
		{"repository root", repo, "", "repository root"},
		{"inside .git", filepath.Join(repo, ".git", "config"), "", "not inside a Git work tree"},
		{"already in the workspace", filepath.Join(ws.Apparatus, "plans", "PLAN.md"), "", "already in the workspace"},
		{"unattached repository", filepath.Join(unlinked, "notes.md"), "", "not attached"},
		{"shelf escapes the space", filepath.Join(repo, "PLAN.md"), "../../elsewhere", "relative path inside"},
		{"missing source", filepath.Join(repo, "nope.md"), "", "does not exist"},
	}
	for _, tc := range refusals {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Stash(tc.src, tc.shelf, true, false)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want %q", err, tc.want)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(repo, "PLAN.md")); err != nil {
		t.Errorf("refused stash must leave the source: %v", err)
	}
}

// Apparatus history must work for anyone: here the global config demands signing with a gpg
// that always fails, and names no identity.
func TestApparatusHistory(t *testing.T) {
	home := isolate(t)
	write(t, filepath.Join(filepath.Dir(home), ".gitconfig"), "[commit]\n\tgpgsign = true\n[gpg]\n\tprogram = false\n")
	repo := newRepo(t, "app", true)
	ws := mustInit(t, repo)
	if got := run(t, ws.Apparatus, "rev-parse", "--show-toplevel"); got != ws.Apparatus {
		t.Fatalf("apparatus is not its own repository: %s", got)
	}

	write(t, filepath.Join(repo, "notes", "idea.md"), "keep me\n")
	st, err := Stash(filepath.Join(repo, "notes"), "research", true, false)
	if err != nil || st.Commit != nil {
		t.Fatalf("stash: %v, commit: %v", err, st.Commit)
	}
	sha := run(t, repo, "rev-parse", "--short", "HEAD")
	if msg := run(t, ws.Apparatus, "log", "-1", "--format=%B"); msg != "Stash notes\n\nCanon: app "+sha {
		t.Errorf("commit message = %q", msg)
	}
	if st := run(t, ws.Apparatus, "status", "--porcelain"); st != "" {
		t.Errorf("stash left apparatus unrecorded: %q", st)
	}

	t.Run("working stashes stay out of history", func(t *testing.T) {
		write(t, filepath.Join(repo, "run.log"), "x\n")
		before := run(t, ws.Apparatus, "rev-parse", "HEAD")
		if _, err := Stash(filepath.Join(repo, "run.log"), "", false, false); err != nil {
			t.Fatal(err)
		}
		if after := run(t, ws.Apparatus, "rev-parse", "HEAD"); after != before {
			t.Error("a working stash committed to apparatus")
		}
	})

	t.Run("an existing unversioned apparatus gets history on the next init", func(t *testing.T) {
		if err := os.RemoveAll(filepath.Join(ws.Apparatus, ".git")); err != nil {
			t.Fatal(err)
		}
		mustInit(t, repo)
		if files := run(t, ws.Apparatus, "ls-files"); files != "research/notes/idea.md" {
			t.Errorf("first commit holds %q", files)
		}
		if n, ok := ws.Unrecorded(); !ok || n != 0 {
			t.Errorf("unrecorded = %d, %v", n, ok)
		}
	})

	t.Run("an apparatus path the human chose is left alone", func(t *testing.T) {
		mine := filepath.Join(t.TempDir(), "notes")
		write(t, filepath.Join(mine, "a.md"), "x\n")
		ws.Manifest.Spaces.Apparatus.Path = mine
		if err := ws.save(); err != nil {
			t.Fatal(err)
		}
		custom := mustInit(t, repo)
		if _, err := os.Stat(filepath.Join(custom.Apparatus, ".git")); !os.IsNotExist(err) {
			t.Errorf("vern initialized a repository in a chosen path: %v", err)
		}
	})
}

// A workspace may physically contain its repository and name it by relative path; no link needed.
func TestPromoteInNestedLayoutWithoutLink(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, "project")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, repo, "init", "-q")
	write(t, filepath.Join(root, ManifestName), `version: 0
spaces:
  canon: {path: ./project}
  apparatus: {path: ./apparatus}
  working: {path: ./working}
`)
	write(t, filepath.Join(root, "working", "notes.md"), "n\n")

	if found, err := Find(repo); err != nil || found.Canon != repo {
		t.Fatalf("Find = %+v, %v", found, err)
	}
	p, err := PlanPromotion(filepath.Join(root, "working", "notes.md"), repo+"/")
	if err != nil {
		t.Fatal(err)
	}
	if p.Dst != filepath.Join(repo, "notes.md") {
		t.Errorf("dst = %s", p.Dst)
	}
}

func TestPromoteScanBlocksSecretsAndLocalPaths(t *testing.T) {
	home := filepath.Dir(isolate(t))
	repo := newRepo(t, "app", true)
	ws := mustInit(t, repo)
	fakeKey := "AK" + "IA" + strings.Repeat("Q", 16) // split so repository scanners ignore it
	write(t, filepath.Join(ws.Working, "notes.md"), "fine\nsee "+filepath.Join(home, "secret", "plan.md")+"\nkey "+fakeKey+"\n")

	p, err := PlanPromotion(filepath.Join(ws.Working, "notes.md"), filepath.Join(repo, "notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Finding{{Rel: ".", Line: 2, Kind: "local path"}, {Rel: ".", Line: 3, Kind: "AWS access key"}}
	if len(p.Findings) != len(want) || p.Findings[0] != want[0] || p.Findings[1] != want[1] {
		t.Fatalf("findings = %+v, want %+v", p.Findings, want)
	}
	if err := p.Apply(false); err == nil {
		t.Fatal("Apply succeeded despite findings")
	}
	if _, err := os.Stat(filepath.Join(repo, "notes.md")); !os.IsNotExist(err) {
		t.Errorf("blocked promotion wrote into canon: %v", err)
	}
	if err := p.Apply(true); err != nil {
		t.Errorf("Apply with allowFindings: %v", err)
	}
}
