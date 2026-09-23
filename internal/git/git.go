package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ── Data types ────────────────────────────────────────────────────────────────

type FileStatus struct {
	Path           string
	StagedStatus   string
	UnstagedStatus string
}

type Commit struct {
	Hash    string
	Short   string
	Message string
	Author  string
	Date    string
	Refs    string
}

type Branch struct {
	Name      string
	IsCurrent bool
	IsRemote  bool
	Upstream  string
}

type RepoStatus struct {
	Branch    string
	Upstream  string
	Ahead     int
	Behind    int
	Staged    []FileStatus
	Unstaged  []FileStatus
	Untracked []FileStatus
	IsClean   bool
}

// Remote represents a configured git remote.
type Remote struct {
	Name     string
	FetchURL string
	PushURL  string
}

// ── Internal exec helper ──────────────────────────────────────────────────────

// commandTimeout bounds how long any single git invocation may run. It's a
// safety net, not a normal-operation limit — large pushes/clones should
// finish well inside it — but it guarantees the TUI can never hang forever
// waiting on a subprocess that's stuck (e.g. a credential helper or ssh
// prompt that ignores the settings below and blocks on a tty read).
const commandTimeout = 2 * time.Minute

func run(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)

	// Never let git or ssh fall back to an interactive credential prompt.
	// gitgetter runs inside a full-screen Bubble Tea program that already
	// owns the terminal, so a prompt has nowhere safe to go: git's own
	// prompt fights bubbletea for stdin, and ssh's prompt (for passphrases
	// or unknown host keys) opens /dev/tty *directly* — bypassing cmd.Stdin
	// entirely — and can scribble raw text into the middle of the UI while
	// stealing keystrokes from it. Disabling prompts turns a hang/corrupted
	// screen into an ordinary error the app already knows how to surface.
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
	)

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return strings.TrimSpace(string(out)),
			fmt.Errorf("git %s timed out after %s (likely waiting on authentication)", args[0], commandTimeout)
	}
	return strings.TrimSpace(string(out)), err
}

// ── Repo detection ────────────────────────────────────────────────────────────

func IsGitRepo() bool {
	_, err := run("rev-parse", "--git-dir")
	return err == nil
}

// ── Status ────────────────────────────────────────────────────────────────────

func GetStatus() (*RepoStatus, error) {
	branch, err := run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("not a git repository")
	}

	out, _ := run("status", "--porcelain=v1")
	status := &RepoStatus{Branch: branch}

	for _, line := range strings.Split(out, "\n") {
		if len(line) < 3 {
			continue
		}
		x := string(line[0])
		y := string(line[1])
		path := strings.TrimSpace(line[3:])
		if idx := strings.Index(path, " -> "); idx != -1 {
			path = path[idx+4:]
		}
		if x != " " && x != "?" {
			status.Staged = append(status.Staged, FileStatus{
				Path:         path,
				StagedStatus: codeToName(x),
			})
		}
		if y != " " && y != "?" {
			status.Unstaged = append(status.Unstaged, FileStatus{
				Path:           path,
				UnstagedStatus: codeToName(y),
			})
		}
		if x == "?" && y == "?" {
			status.Untracked = append(status.Untracked, FileStatus{
				Path:           path,
				UnstagedStatus: "untracked",
			})
		}
	}

	status.IsClean = len(status.Staged) == 0 && len(status.Unstaged) == 0 && len(status.Untracked) == 0

	upstream, err := run("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err == nil && !strings.Contains(upstream, "fatal") {
		status.Upstream = upstream
		ab, err := run("rev-list", "--count", "--left-right", "@{u}...HEAD")
		if err == nil {
			parts := strings.Fields(ab)
			if len(parts) == 2 {
				status.Behind, _ = strconv.Atoi(parts[0])
				status.Ahead, _ = strconv.Atoi(parts[1])
			}
		}
	}
	return status, nil
}

func codeToName(code string) string {
	switch code {
	case "M":
		return "modified"
	case "A":
		return "added"
	case "D":
		return "deleted"
	case "R":
		return "renamed"
	case "C":
		return "copied"
	case "U":
		return "unmerged"
	default:
		return code
	}
}

// ── Log ───────────────────────────────────────────────────────────────────────

func GetLog(n int) ([]Commit, error) {
	format := "%H|%h|%s|%an|%cr|%D"
	out, err := run("log", fmt.Sprintf("-n%d", n), "--pretty=format:"+format)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var commits []Commit
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 6)
		if len(parts) < 5 {
			continue
		}
		c := Commit{Hash: parts[0], Short: parts[1], Message: parts[2], Author: parts[3], Date: parts[4]}
		if len(parts) == 6 {
			c.Refs = parts[5]
		}
		commits = append(commits, c)
	}
	return commits, nil
}

// ── Branches ──────────────────────────────────────────────────────────────────

func GetBranches() ([]Branch, error) {
	// %(refname) (the long form, e.g. "refs/remotes/origin/main") is what
	// actually tells local and remote-tracking refs apart — %(refname:short)
	// gives "origin/main" for both a real local branch named that and a
	// remote-tracking ref, so a prefix check on the short name (the old
	// approach) can never work.
	out, err := run("branch", "-a", "--format=%(HEAD)|%(refname)|%(refname:short)|%(upstream:short)")
	if err != nil {
		return nil, err
	}
	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 3 {
			continue
		}
		fullRef, short := parts[1], parts[2]

		// "origin/HEAD" is a symbolic alias for another entry already in
		// this list (e.g. "origin/main") — it's not a real branch, and
		// checking it out by name is meaningless, so skip it.
		if strings.HasSuffix(fullRef, "/HEAD") {
			continue
		}

		b := Branch{
			Name:      short,
			IsCurrent: parts[0] == "*",
			IsRemote:  strings.HasPrefix(fullRef, "refs/remotes/"),
		}
		if len(parts) == 4 {
			b.Upstream = parts[3]
		}
		branches = append(branches, b)
	}
	return branches, nil
}

// LocalNameForRemote strips the leading "<remote>/" from a remote-tracking
// branch's short name (e.g. "origin/feature-x" -> "feature-x"). Passing the
// result to CheckoutBranch lets git's own DWIM behavior create (or switch
// to) a local tracking branch, instead of checking out the remote ref
// directly and leaving HEAD detached.
func LocalNameForRemote(remoteShortName string) string {
	if idx := strings.Index(remoteShortName, "/"); idx != -1 {
		return remoteShortName[idx+1:]
	}
	return remoteShortName
}

// ── Remotes ───────────────────────────────────────────────────────────────────

func GetRemotes() ([]Remote, error) {
	out, err := run("remote", "-v")
	if err != nil || out == "" {
		return nil, err
	}
	seen := map[string]*Remote{}
	var order []string
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		name, url, typ := parts[0], parts[1], strings.Trim(parts[2], "()")
		if _, ok := seen[name]; !ok {
			seen[name] = &Remote{Name: name}
			order = append(order, name)
		}
		if typ == "fetch" {
			seen[name].FetchURL = url
		} else if typ == "push" {
			seen[name].PushURL = url
		}
	}
	remotes := make([]Remote, 0, len(order))
	for _, name := range order {
		remotes = append(remotes, *seen[name])
	}
	return remotes, nil
}

func AddRemote(name, url string) error {
	_, err := run("remote", "add", name, url)
	return err
}

func RemoveRemote(name string) error {
	out, err := run("remote", "remove", name)
	if err != nil {
		return fmt.Errorf("%s", out)
	}
	return nil
}

func RenameRemote(oldName, newName string) error {
	_, err := run("remote", "rename", oldName, newName)
	return err
}

func SetRemoteURL(name, url string) error {
	_, err := run("remote", "set-url", name, url)
	return err
}

// ── Staging ───────────────────────────────────────────────────────────────────

func StageFile(path string) error {
	_, err := run("add", "--", path)
	return err
}

func UnstageFile(path string) error {
	_, err := run("restore", "--staged", "--", path)
	return err
}

func StageAll() error {
	_, err := run("add", "-A")
	return err
}

// ── Commits ───────────────────────────────────────────────────────────────────

// CreateCommit creates a commit with the given message. When signoff is true,
// a Signed-off-by trailer is added via `git commit --signoff`.
func CreateCommit(message string, signoff bool) error {
	args := []string{"commit", "-m", message}
	if signoff {
		args = append(args, "--signoff")
	}
	_, err := run(args...)
	return err
}

// AmendCommit amends HEAD. When signoff is true, `--signoff` is added.
func AmendCommit(message string, signoff bool) error {
	args := []string{"commit", "--amend"}
	if message == "" {
		args = append(args, "--no-edit")
	} else {
		args = append(args, "-m", message)
	}
	if signoff {
		args = append(args, "--signoff")
	}
	_, err := run(args...)
	return err
}

func SquashCommits(n int, message string, signoff bool) error {
	if _, err := run("reset", "--soft", fmt.Sprintf("HEAD~%d", n)); err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}
	if err := CreateCommit(message, signoff); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}
	return nil
}

// ── Remote ops ────────────────────────────────────────────────────────────────

func Push(forceLease bool) (string, error) {
	args := []string{"push"}
	if forceLease {
		args = append(args, "--force-with-lease")
	}
	return run(args...)
}

func Pull() (string, error) {
	return run("pull")
}

func Fetch() (string, error) {
	return run("fetch", "--all", "--prune")
}

// ── Branch ops ────────────────────────────────────────────────────────────────

func CheckoutBranch(name string) error {
	_, err := run("checkout", name)
	return err
}

func CreateBranch(name string) error {
	_, err := run("checkout", "-b", name)
	return err
}

func DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	out, err := run("branch", flag, name)
	if err != nil {
		return fmt.Errorf("%s", out)
	}
	return nil
}

func RenameBranch(oldName, newName string) error {
	_, err := run("branch", "-m", oldName, newName)
	return err
}

// ── Stash ─────────────────────────────────────────────────────────────────────

func Stash(message string) error {
	if message == "" {
		_, err := run("stash")
		return err
	}
	_, err := run("stash", "push", "-m", message)
	return err
}

func StashPop() error {
	out, err := run("stash", "pop")
	if err != nil {
		return fmt.Errorf("%s", out)
	}
	return nil
}

func StashDrop(index int) error {
	out, err := run("stash", "drop", fmt.Sprintf("stash@{%d}", index))
	if err != nil {
		return fmt.Errorf("%s", out)
	}
	return nil
}

func GetStashes() ([]string, error) {
	out, err := run("stash", "list")
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}
