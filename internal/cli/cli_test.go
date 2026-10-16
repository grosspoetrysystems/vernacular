package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type result struct {
	code        int
	out, errOut string
}

func vern(t *testing.T, stdin string, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, strings.NewReader(stdin), &out, &errOut)
	return result{code, out.String(), errOut.String()}
}

// setup isolates HOME, creates a committed repository, chdirs into it, and returns its real path.
func setup(t *testing.T) string {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "root"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	t.Chdir(repo)
	return repo
}

func mustRun(t *testing.T, args ...string) string {
	t.Helper()
	r := vern(t, "", args...)
	if r.code != 0 {
		t.Fatalf("vern %v exited %d: %s", args, r.code, r.errOut)
	}
	return strings.TrimSpace(r.out)
}

func TestExitCodes(t *testing.T) {
	setup(t)
	mustRun(t, "init")
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"no command", nil, 2},
		{"unknown command", []string{"bogus"}, 2},
		{"help", []string{"help"}, 0},
		{"path without space", []string{"path"}, 2},
		{"path unknown space", []string{"path", "attic"}, 2},
		{"promote missing DEST", []string{"promote", "x"}, 2},
		{"stash missing SRC", []string{"stash"}, 2},
		{"stash extra argument", []string{"stash", "a", "b", "c"}, 2},
		{"stash undo with SRC and SHELF", []string{"stash", "--undo", "a", "b"}, 2},
		{"stash undo with --apparatus", []string{"stash", "--undo", "--apparatus"}, 2},
		{"unknown flag", []string{"status", "--verbose"}, 2},
		{"command help flag", []string{"promote", "-h"}, 0},
		{"operational failure", []string{"attach", "vrn_00000000"}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if r := vern(t, "", tc.args...); r.code != tc.want {
				t.Errorf("exit %d, want %d; stderr: %s", r.code, tc.want, r.errOut)
			}
		})
	}
	if r := vern(t, "", "status", "--verbose"); strings.Count(r.errOut, "not defined") != 1 {
		t.Errorf("flag error should be reported once:\n%s", r.errOut)
	}
}

func TestInitReportsAbsoluteCanonAndPathResolves(t *testing.T) {
	repo := setup(t)
	out := mustRun(t, "init")
	if !strings.Contains(out, "  canon      "+repo+"\n") {
		t.Errorf("init should print canon absolutely, got:\n%s", out)
	}
	if got := mustRun(t, "path", "canon"); got != repo {
		t.Errorf("path canon = %q, want %q", got, repo)
	}
	working := mustRun(t, "path", "working")
	if fi, err := os.Stat(working); err != nil || !fi.IsDir() || filepath.Base(working) != "working" {
		t.Errorf("path working = %q (%v)", working, err)
	}
	if out := mustRun(t, "status"); !strings.Contains(out, "Canon  "+repo) {
		t.Errorf("status:\n%s", out)
	}
}

func TestStatusListsNewestFilesFirstWithinLimit(t *testing.T) {
	setup(t)
	mustRun(t, "init")
	apparatus := mustRun(t, "path", "apparatus")
	base := time.Now().Add(-72 * time.Hour)
	for i := range recentLimit + 2 {
		p := filepath.Join(apparatus, "notes", fmt.Sprintf("n%02d.md", i))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		mod := base.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(p, mod, mod); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(apparatus, ".DS_Store"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	out := mustRun(t, "status")
	newest, oldestListed, firstHidden := "notes/n11.md", "notes/n02.md", "notes/n01.md"
	if i, j := strings.Index(out, newest), strings.Index(out, oldestListed); i < 0 || j < 0 || i > j {
		t.Errorf("want %s listed before %s:\n%s", newest, oldestListed, out)
	}
	if strings.Contains(out, firstHidden) || strings.Contains(out, ".DS_Store") {
		t.Errorf("listed past the limit or a dotfile:\n%s", out)
	}
	if !strings.Contains(out, "13 files, 1 directory") || !strings.Contains(out, "3 more files not listed") {
		t.Errorf("counts or remainder wrong:\n%s", out)
	}
}

func TestPathCanonUnknownExplainsItself(t *testing.T) {
	setup(t)
	dir := t.TempDir()
	manifest := "version: 0\nspaces:\n  apparatus: {path: ./a}\n  working: {path: ./w}\n"
	if err := os.WriteFile(filepath.Join(dir, "vernacular.yml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	r := vern(t, "", "path", "canon")
	if r.code != 1 || !strings.Contains(r.errOut, "canon is unknown here") {
		t.Errorf("exit %d, stderr %q", r.code, r.errOut)
	}
}

func TestPromoteArgumentHandling(t *testing.T) {
	repo := setup(t)
	mustRun(t, "init")
	working := mustRun(t, "path", "working")
	src := filepath.Join(working, "notes.md")
	if err := os.WriteFile(src, []byte("notes\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for i, args := range [][]string{
		{"promote", "--yes", src, "a/"},
		{"promote", src, "b/", "--yes"},
		{"promote", "-y", "--", src, "c/"},
	} {
		if r := vern(t, "", args...); r.code != 0 {
			t.Errorf("case %d %v exited %d: %s", i, args, r.code, r.errOut)
		}
	}
	for _, d := range []string{"a", "b", "c"} {
		if _, err := os.Stat(filepath.Join(repo, d, "notes.md")); err != nil {
			t.Errorf("%s/notes.md not promoted: %v", d, err)
		}
	}

	// After "--", --yes is a positional argument, not a flag.
	if r := vern(t, "", "promote", "--", src, "d/", "--yes"); r.code != 2 {
		t.Errorf("--yes after -- should be positional; exit %d", r.code)
	}

	if r := vern(t, "n\n", "promote", src, "e/"); r.code != 1 || !strings.Contains(r.errOut, "cancelled") {
		t.Errorf("declined promote: exit %d, stderr %q", r.code, r.errOut)
	}
	if _, err := os.Stat(filepath.Join(repo, "e")); !os.IsNotExist(err) {
		t.Errorf("declined promote wrote into canon: %v", err)
	}
	if r := vern(t, "y\n", "promote", src, "e/"); r.code != 0 {
		t.Errorf("confirmed promote: exit %d, stderr %q", r.code, r.errOut)
	}
}

func TestStashUndoRestoresByDefaultAndByID(t *testing.T) {
	repo := setup(t)
	mustRun(t, "init")
	for _, name := range []string{"it's notes", "later"} {
		if err := os.MkdirAll(filepath.Join(repo, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, name, "a.md"), []byte(name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	first := mustRun(t, "stash", "it's notes", "research")
	kept := mustRun(t, "stash", "--apparatus", "later")
	head, err := exec.Command("git", "-C", mustRun(t, "path", "apparatus"), "rev-parse", "--short", "HEAD").Output()
	if err != nil || !strings.Contains(kept, "Saved in apparatus history ("+strings.TrimSpace(string(head))+")") {
		t.Errorf("apparatus stash did not name its save point %s:\n%s", head, kept)
	}
	id := first[strings.Index(first, "--undo ")+len("--undo "):]

	// By ID: the older stash comes back while the newer one stays put.
	mustRun(t, "stash", "--undo", id)
	if got, err := os.ReadFile(filepath.Join(repo, "it's notes", "a.md")); err != nil || string(got) != "it's notes\n" {
		t.Fatalf("undo by ID did not restore: %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(repo, "later")); !os.IsNotExist(err) {
		t.Fatalf("undo by ID touched another stash: %v", err)
	}
	// Default: the latest remaining stash.
	mustRun(t, "stash", "--undo")
	if _, err := os.Stat(filepath.Join(repo, "later", "a.md")); err != nil {
		t.Fatalf("undo did not restore the latest stash: %v", err)
	}
	if r := vern(t, "", "stash", "--undo"); r.code != 1 || !strings.Contains(r.errOut, "nothing to undo") {
		t.Errorf("undo with everything undone: exit %d, %q", r.code, r.errOut)
	}
	if r := vern(t, "", "stash", "--undo", id); r.code != 1 || !strings.Contains(r.errOut, "already undone") {
		t.Errorf("undo of an undone ID: exit %d, %q", r.code, r.errOut)
	}
	if r := vern(t, "", "stash", "--undo", "st_nope"); r.code != 1 || !strings.Contains(r.errOut, "no stash st_nope") {
		t.Errorf("undo of an unknown ID: exit %d, %q", r.code, r.errOut)
	}
	// The journal keeps both stashes, each marked undone.
	journal, err := os.ReadFile(filepath.Join(filepath.Dir(mustRun(t, "path", "working")), "stash.jsonl"))
	if err != nil || strings.Count(string(journal), "\n") != 2 || strings.Count(string(journal), `"undone_at"`) != 2 {
		t.Errorf("journal after two undos:\n%s", journal)
	}
}

func TestStashUndoNeverOverwrites(t *testing.T) {
	repo := setup(t)
	mustRun(t, "init")
	if err := os.WriteFile(filepath.Join(repo, "PLAN.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "stash", "PLAN.md")
	if err := os.WriteFile(filepath.Join(repo, "PLAN.md"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := vern(t, "", "stash", "--undo"); r.code != 1 || !strings.Contains(r.errOut, "never overwrites") {
		t.Errorf("exit %d, %q", r.code, r.errOut)
	}
	if got, _ := os.ReadFile(filepath.Join(repo, "PLAN.md")); string(got) != "new\n" {
		t.Errorf("undo overwrote canon: %q", got)
	}
	if _, err := os.Stat(filepath.Join(mustRun(t, "path", "working"), "PLAN.md")); err != nil {
		t.Errorf("refused undo lost the stashed copy: %v", err)
	}
}

func TestAttachListing(t *testing.T) {
	repo := setup(t)
	if out := mustRun(t, "attach"); !strings.Contains(out, "No workspaces registered") {
		t.Errorf("empty listing: %q", out)
	}
	mustRun(t, "init")
	id := strings.TrimSpace(strings.SplitN(mustRun(t, "status"), "  ", 3)[1])

	home, _ := os.UserHomeDir()
	if err := os.WriteFile(filepath.Join(home, ".vern", "workspaces", "vrn_deadbeef"), []byte("/nonexistent\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := mustRun(t, "attach")
	if !strings.Contains(out, id) || !strings.Contains(out, "last seen with "+repo) {
		t.Errorf("listing lacks %s with %s:\n%s", id, repo, out)
	}
	if !strings.Contains(out, "vrn_deadbeef  (broken:") {
		t.Errorf("listing lacks broken entry:\n%s", out)
	}
}
