package app

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gitgetter/internal/git"
	"gitgetter/internal/styles"
)

// ── View tabs ─────────────────────────────────────────────────────────────────

type ViewTab int

const (
	TabStatus ViewTab = iota
	TabLog
	TabBranch
	TabRemote
	TabStash
	tabCount
)

var tabMeta = [tabCount]struct{ icon, name string }{
	TabStatus: {" ", "Status"},
	TabLog:    {" ", "Log"},
	TabBranch: {" ", "Branches"},
	TabRemote: {" ", "Remote"},
	TabStash:  {" ", "Stash"},
}

// ── Remote actions ────────────────────────────────────────────────────────────

type remoteAction struct {
	icon, label, desc string
}

var remoteActions = []remoteAction{
	{" ", "Push", "Send local commits to the remote"},
	{" ", "Pull", "Fetch and merge remote changes"},
	{" ", "Fetch", "Download remote refs without merging"},
	{"󰘍 ", "Force Push", "Push with lease — rewrites remote history safely"},
}

// ── Input modes ───────────────────────────────────────────────────────────────

type InputMode int

const (
	ModeNormal InputMode = iota
	ModeCommit
	ModeAmend
	ModeBranch
	ModeSquash
	ModeStash
	ModeRename
	ModeAddRemote
	ModeRemoteURL
	ModeEditRemote
)

// ── Notifications ─────────────────────────────────────────────────────────────

type NotifyLevel int

const (
	NotifySuccess NotifyLevel = iota
	NotifyError
	NotifyInfo
)

type Notification struct {
	Message string
	Level   NotifyLevel
	Expiry  time.Time
}

// ── Tea messages ─────────────────────────────────────────────────────────────

type (
	msgStatus struct{ v *git.RepoStatus }
	msgLog    struct{ v []git.Commit }
	msgBranch struct{ v []git.Branch }
	msgStash  struct{ v []string }
	msgRemote struct{ v []git.Remote }
	msgErr    struct{ err error }
	msgNotify struct {
		text  string
		level NotifyLevel
	}
	msgClearNote    struct{}
	msgStageAllDone struct{}
)

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	width, height int

	tab      ViewTab
	showHelp bool // true = full-screen help dialog visible

	// Git data
	status   *git.RepoStatus
	commits  []git.Commit
	branches []git.Branch
	stashes  []string
	remotes  []git.Remote

	cursor   [tabCount]int
	selected [tabCount]map[int]bool

	// Input overlay
	inputMode  InputMode
	inputLabel string
	inputHint  string
	textInput  textinput.Model

	// Multi-step operation context
	pendingRemoteName string
	renamingBranch    string
	renamingRemote    string

	// Confirm dialog
	confirmMsg    string
	confirmAction func() tea.Cmd

	// Notification (replaces bottom bar when set)
	note *Notification

	spinner spinner.Model
	loading bool
}

// ── Initializer ──────────────────────────────────────────────────────────────

func InitialModel() Model {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(styles.AccentPurple)

	ti := textinput.New()
	ti.CharLimit = 300
	ti.Width = 64
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.AccentBlue)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.ColorText)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.AccentYellow)

	m := Model{spinner: sp, textInput: ti, loading: true}
	for i := range m.selected {
		m.selected[i] = make(map[int]bool)
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, doRefreshAll())
}

// ── Async commands ────────────────────────────────────────────────────────────

func doRefreshAll() tea.Cmd {
	return tea.Batch(doLoadStatus(), doLoadLog(), doLoadBranches(), doLoadStashes(), doLoadRemotes())
}

func doLoadStatus() tea.Cmd {
	return func() tea.Msg {
		v, err := git.GetStatus()
		if err != nil {
			return msgErr{err}
		}
		return msgStatus{v}
	}
}

func doLoadLog() tea.Cmd {
	return func() tea.Msg {
		v, err := git.GetLog(60)
		if err != nil {
			return msgLog{nil}
		}
		return msgLog{v}
	}
}

func doLoadBranches() tea.Cmd {
	return func() tea.Msg {
		v, err := git.GetBranches()
		if err != nil {
			return msgBranch{nil}
		}
		return msgBranch{v}
	}
}

func doLoadStashes() tea.Cmd {
	return func() tea.Msg {
		v, err := git.GetStashes()
		if err != nil {
			return msgStash{nil}
		}
		return msgStash{v}
	}
}

func doLoadRemotes() tea.Cmd {
	return func() tea.Msg {
		v, err := git.GetRemotes()
		if err != nil {
			return msgRemote{nil}
		}
		return msgRemote{v}
	}
}

func notify(text string, level NotifyLevel) tea.Cmd {
	return func() tea.Msg { return msgNotify{text, level} }
}

func clearNoteAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return msgClearNote{} })
}

// ── Visible tabs ──────────────────────────────────────────────────────────────

func (m Model) visibleTabs() []ViewTab {
	tabs := []ViewTab{TabStatus, TabLog, TabBranch, TabRemote}
	if len(m.stashes) > 0 {
		tabs = append(tabs, TabStash)
	}
	return tabs
}

// ── Layout helpers ────────────────────────────────────────────────────────────

// keybindPanelW returns the width of the right-side keybind panel (0 if narrow terminal).
func (m Model) keybindPanelW() int {
	if m.width < 90 {
		return 0
	}
	return 26
}

// contentW returns the body content area width (total minus keybind panel).
func (m Model) contentW() int {
	return m.width - m.keybindPanelW()
}

// notifyPanelH returns rows used by the notification panel (0 or 2).
func (m Model) notifyPanelH() int {
	if m.note == nil {
		return 0
	}
	return 2
}

// bodyHeight returns available rows for body content.
// Layout: tabBar(1) + tabBorder(1) + body + (bottomBar(1) OR notification(2))
func (m Model) bodyHeight() int {
	const chrome = 2 // tab labels row + tab border row
	foot := 1        // bottom bar
	if m.note != nil {
		foot = m.notifyPanelH()
	}
	h := m.height - chrome - foot
	if h < 1 {
		h = 1
	}
	return h
}

// ── Cursor / selection helpers ────────────────────────────────────────────────

func (m *Model) cur() int         { return m.cursor[m.tab] }
func (m *Model) setCur(n int)     { m.cursor[m.tab] = n }
func (m *Model) isSel(i int) bool { return m.selected[m.tab][i] }
func (m *Model) selCount() int    { return len(m.selected[m.tab]) }
func (m *Model) clearSel()        { m.selected[m.tab] = make(map[int]bool) }

func (m *Model) toggleSel(i int) {
	if m.selected[m.tab][i] {
		delete(m.selected[m.tab], i)
	} else {
		m.selected[m.tab][i] = true
	}
}

func (m *Model) listLen() int {
	switch m.tab {
	case TabStatus:
		if m.status == nil {
			return 0
		}
		return len(m.status.Staged) + len(m.status.Unstaged) + len(m.status.Untracked)
	case TabLog:
		return len(m.commits)
	case TabBranch:
		return len(m.branches)
	case TabRemote:
		return len(remoteActions) + len(m.remotes) + 1
	case TabStash:
		return len(m.stashes)
	}
	return 0
}

func (m *Model) clampCur() {
	n := m.listLen()
	if n == 0 {
		m.setCur(0)
		return
	}
	c := m.cur()
	if c < 0 {
		m.setCur(0)
	} else if c >= n {
		m.setCur(n - 1)
	}
}

func (m *Model) allStatusFiles() []git.FileStatus {
	if m.status == nil {
		return nil
	}
	out := make([]git.FileStatus, 0,
		len(m.status.Staged)+len(m.status.Unstaged)+len(m.status.Untracked))
	out = append(out, m.status.Staged...)
	out = append(out, m.status.Unstaged...)
	out = append(out, m.status.Untracked...)
	return out
}

func (m *Model) selectedFilePaths() []string {
	files := m.allStatusFiles()
	var paths []string
	for idx := range m.selected[TabStatus] {
		if idx < len(files) {
			paths = append(paths, files[idx].Path)
		}
	}
	return paths
}

func (m *Model) cursorFilePath() string {
	files := m.allStatusFiles()
	c := m.cur()
	if c < len(files) {
		return files[c].Path
	}
	return ""
}

func (m *Model) isStaged(idx int) bool {
	if m.status == nil {
		return false
	}
	return idx < len(m.status.Staged)
}

func (m *Model) openInput(mode InputMode, label, placeholder string) {
	m.inputMode = mode
	m.inputLabel = label
	m.inputHint = placeholder
	m.textInput.Reset()
	m.textInput.Placeholder = placeholder
	m.textInput.Focus()
}

// bodyClickItem maps a body-relative click row to a list item index.
// bodyStartRow is 2 (tab + tab-border above, bottom bar below).
func (m Model) bodyClickItem(relRow int) int {
	bH := m.bodyHeight()
	listLen := m.listLen()
	if listLen == 0 || bH <= 0 || relRow < 0 {
		return -1
	}

	linesPerItem := 1
	cursorLine := m.cursor[m.tab]
	if m.tab == TabLog {
		linesPerItem = 2
		cursorLine = m.cursor[m.tab] * 2
	}

	totalLines := listLen * linesPerItem
	start := cursorLine - bH/2
	if start < 0 {
		start = 0
	}
	end := start + bH
	if end > totalLines {
		end = totalLines
		start = end - bH
		if start < 0 {
			start = 0
		}
	}

	clicked := start + relRow
	if clicked < 0 || clicked >= totalLines {
		return -1
	}
	item := clicked / linesPerItem
	if item >= listLen {
		return listLen - 1
	}
	return item
}
