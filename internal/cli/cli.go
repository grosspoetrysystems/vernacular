// Package cli is the vern command dispatcher.
package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/grosspoetrysystems/vernacular/internal/workspace"
)

const usage = `vern — canon, apparatus, and working spaces around a Git repository

Usage:
  vern init                            attach this repository to a new private workspace
  vern attach [ID | DIR]               attach this repository to an existing workspace;
                                       without arguments, list workspaces
  vern path canon|apparatus|working    print a space's absolute path
  vern status                          show the workspace and what exists outside canon
  vern promote [--yes] [--allow-findings] SRC DEST
                                       copy SRC from apparatus/working to DEST in canon

init creates ~/.vern/data/<id>/ with vernacular.yml, apparatus/, and working/,
registers it in ~/.vern/workspaces/<id>, and stores the id in the repository's
local Git config as vernacular.workspace. Tracked files are never touched. The
attachment follows the repository when it moves and is shared by worktrees.

attach DIR re-registers a workspace directory, recovering one that moved or a
lost ~/.vern. To detach: git config --unset vernacular.workspace

promote works from any directory: SRC must be inside a workspace's apparatus or
working, and DEST inside a repository attached to that workspace. It copies; it
never overwrites, stages, or commits. It refuses destinations canon ignores,
refuses files that look like secrets or local paths unless --allow-findings is
given, and asks for confirmation unless --yes is given.
`

// Run executes vern with args (excluding the program name) and returns the exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "init":
		err = runInit(args[1:], stdout)
	case "attach":
		err = runAttach(args[1:], stdout)
	case "path":
		err = runPath(args[1:], stdout)
	case "status":
		err = runStatus(args[1:], stdout)
	case "promote":
		err = runPromote(args[1:], stdin, stdout)
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stdout, usage)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "vern: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
	var ue usageError
	switch {
	case errors.As(err, &ue):
		_, _ = fmt.Fprintf(stderr, "vern %s: %v\n\n%s", args[0], err, usage)
		return 2
	case errors.Is(err, flag.ErrHelp):
		_, _ = fmt.Fprint(stdout, usage)
		return 0
	case err != nil:
		_, _ = fmt.Fprintf(stderr, "vern: %v\n", err)
		return 1
	}
	return 0
}

type usageError string

func (e usageError) Error() string { return string(e) }

// parse allows flags before, between, or after positional arguments. Everything after the
// first "--" is positional.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos, rest []string
	for i, a := range args {
		if a == "--" {
			args, rest = args[:i], args[i+1:]
			break
		}
	}
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, usageError(err.Error())
		}
		if fs.NArg() == 0 {
			return append(pos, rest...), nil
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func runInit(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return usageError("init takes no arguments; run it inside the repository")
	}
	res, err := workspace.Init(".")
	if err != nil {
		return err
	}
	ws := res.Workspace
	if res.Existing {
		_, _ = fmt.Fprintf(stdout, "Already attached to %s at %s\n", ws.ID(), workspace.Abbrev(ws.Root))
		return nil
	}
	_, _ = fmt.Fprintf(stdout, "Created workspace %s at %s\n", ws.ID(), workspace.Abbrev(ws.Root))
	printSpaces(stdout, ws)
	_, _ = fmt.Fprintf(stdout, "\nOptional: add this to the repository's AGENTS.md (or CLAUDE.md):\n\n%s", agentSnippet)
	return nil
}

func runAttach(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("attach", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	switch len(pos) {
	case 0:
		list, err := workspace.List()
		if err != nil {
			return err
		}
		if len(list) == 0 {
			_, _ = fmt.Fprintln(stdout, "No workspaces registered. Run vern init inside a repository.")
		}
		for _, e := range list {
			switch {
			case e.Err != nil:
				_, _ = fmt.Fprintf(stdout, "%s  (broken: %s)\n", e.ID, strings.SplitN(e.Err.Error(), "\n", 2)[0])
			case e.Canon != "":
				_, _ = fmt.Fprintf(stdout, "%s  %s  last seen with %s\n", e.ID, workspace.Abbrev(e.Root), workspace.Abbrev(e.Canon))
			default:
				_, _ = fmt.Fprintf(stdout, "%s  %s\n", e.ID, workspace.Abbrev(e.Root))
			}
		}
		return nil
	case 1:
	default:
		return usageError("expected one ID or DIR")
	}
	ws, previous, err := workspace.Attach(".", pos[0])
	if err != nil {
		return err
	}
	if previous != "" && previous != ws.ID() {
		_, _ = fmt.Fprintf(stdout, "Detached from %s (its files are untouched)\n", previous)
	}
	_, _ = fmt.Fprintf(stdout, "Attached to %s at %s\n", ws.ID(), workspace.Abbrev(ws.Root))
	printSpaces(stdout, ws)
	return nil
}

func runPath(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("path", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usageError("expected one of canon, apparatus, working")
	}
	ws, err := workspace.Find(".")
	if err != nil {
		return err
	}
	p := map[string]string{"canon": ws.Canon, "apparatus": ws.Apparatus, "working": ws.Working}[pos[0]]
	switch {
	case pos[0] != "canon" && pos[0] != "apparatus" && pos[0] != "working":
		return usageError("unknown space " + pos[0])
	case p == "":
		return errors.New("canon is unknown here; run vern path canon inside the repository")
	}
	_, _ = fmt.Fprintln(stdout, p)
	return nil
}

func printSpaces(w io.Writer, ws *workspace.Workspace) {
	_, _ = fmt.Fprintf(w, "  canon      %s\n  apparatus  %s\n  working    %s\n",
		workspace.Abbrev(ws.Canon), workspace.Abbrev(ws.Apparatus), workspace.Abbrev(ws.Working))
}

// agentSnippet carries no paths: workspace locations are private and machine-specific.
const agentSnippet = `## Vernacular workspace

This repository is project canon. Its working context lives outside it:
product documents, decisions, research, experiments, and earlier agents'
reports. Before planning work or reporting on status or direction, list
and read what is relevant under ` + "`vern path apparatus`" + ` and ` + "`vern path working`" + `.

Do not create exploratory reports, plans, research, benchmark output,
or temporary Markdown in this repository unless explicitly requested.

For temporary research, generated reports, and scratch files, write under
the path printed by ` + "`vern path working`" + `.
For durable non-canonical work, use ` + "`vern path apparatus`" + `.

Only move material into this repository through an intentional
project change, such as ` + "`vern promote`" + `.
`

func runStatus(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return usageError("status takes no arguments")
	}
	ws, err := workspace.Find(".")
	if err != nil {
		return err
	}
	id := ws.ID()
	if id == "" {
		id = "(hand-written manifest)"
	}
	_, _ = fmt.Fprintf(stdout, "Workspace  %s  %s\n\n", id, ws.Root)
	if ws.Canon != "" {
		_, _ = fmt.Fprintf(stdout, "Canon  %s\n", ws.Canon)
	} else {
		_, _ = fmt.Fprintln(stdout, "Canon  unknown here (run vern status inside the repository)")
	}
	now := time.Now()
	for _, s := range []struct{ name, label, dir string }{
		{"apparatus", ws.Apparatus, ws.Apparatus},
		{"working", ws.Working, ws.Working},
	} {
		tr, err := workspace.ScanTree(s.dir, recentLimit)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "\n%s%s  %s\n", strings.ToUpper(s.name[:1]), s.name[1:], s.label)
		if tr.Missing {
			_, _ = fmt.Fprintln(stdout, "  missing")
			continue
		}
		_, _ = fmt.Fprintf(stdout, "  %s, %s\n", plural(tr.Files, "file"), plural(tr.Dirs, "directory"))
		for _, it := range tr.Recent {
			_, _ = fmt.Fprintf(stdout, "  %8s  %s\n", age(now.Sub(it.Mod)), filepath.ToSlash(it.Path))
		}
		if more := tr.Files - len(tr.Recent); more > 0 && len(tr.Recent) == recentLimit {
			_, _ = fmt.Fprintf(stdout, "  %8s  … %s not listed\n", "", plural(more, "more file"))
		}
	}
	return nil
}

// recentLimit bounds how many files status lists per space: enough to orient an agent, short
// enough to read at a glance.
const recentLimit = 10

func age(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func runPromote(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	yes := fs.Bool("yes", false, "promote without asking for confirmation")
	fs.BoolVar(yes, "y", false, "shorthand for --yes")
	allow := fs.Bool("allow-findings", false, "promote despite scan findings (they are shown either way)")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 2 {
		return usageError("expected SRC and DEST")
	}
	plan, err := workspace.PlanPromotion(pos[0], pos[1])
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "Promote %s\n     → %s\n\n", workspace.Display(plan.Src), workspace.Display(plan.Dst))
	for _, e := range plan.Entries {
		switch {
		case e.Dir:
			_, _ = fmt.Fprintf(stdout, "  %s/\n", filepath.ToSlash(filepath.Join(filepath.Base(plan.Dst), e.Rel)))
		default:
			_, _ = fmt.Fprintf(stdout, "  %s  (%d bytes)\n", filepath.ToSlash(filepath.Join(filepath.Base(plan.Dst), e.Rel)), e.Size)
		}
	}
	if len(plan.Findings) > 0 {
		_, _ = fmt.Fprintln(stdout, "\nScan found possible secrets or local paths:")
		for _, f := range plan.Findings {
			_, _ = fmt.Fprintf(stdout, "  %s:%d  %s\n", filepath.ToSlash(filepath.Join(workspace.Display(plan.Src), f.Rel)), f.Line, f.Kind)
		}
		if !*allow {
			_, _ = fmt.Fprintln(stdout, "\nRemove them from the source, or rerun with --allow-findings if they are intended.")
			return errors.New("promotion blocked by scan findings")
		}
	}
	_, _ = fmt.Fprintf(stdout, "\n%s will be copied into canon. Nothing is staged or committed.\n", plural(plan.Files(), "file"))

	if !*yes {
		_, _ = fmt.Fprint(stdout, "Promote? [y/N] ")
		answer, _ := bufio.NewReader(stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			if !strings.HasSuffix(answer, "\n") {
				_, _ = fmt.Fprintln(stdout)
			}
			return errors.New("promotion cancelled")
		}
	}
	if err := plan.Apply(*allow); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "Promoted. Review with Git:\n")
	st, err := workspace.Git(plan.Workspace.Canon, "status", "--short", "--untracked-files=all", "--", plan.Dst)
	if err != nil {
		return err
	}
	if st == "" {
		st = "(no changes reported; DEST may be ignored by .gitignore)"
	}
	_, _ = fmt.Fprintf(stdout, "%s\n", st)
	return nil
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	if strings.HasSuffix(noun, "y") {
		return fmt.Sprintf("%d %sies", n, noun[:len(noun)-1])
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
