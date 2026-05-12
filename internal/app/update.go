package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"gitgetter/internal/git"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	{
		sp, cmd := m.spinner.Update(msg)
		m.spinner = sp
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	// ── Mouse ─────────────────────────────────────────────────────────────────
	case tea.MouseMsg:
		nm, cmd := m.handleMouseMsg(msg)
		return nm, tea.Batch(append(cmds, cmd)...)

	// ── Data responses ────────────────────────────────────────────────────────
	case msgStatus:
		m.status = msg.v
		m.loading = false
		m.clampCur()

	case msgLog:
		m.commits = msg.v
		if m.tab == TabLog {
			m.clampCur()
		}

	case msgBranch:
		m.branches = msg.v
		if m.tab == TabBranch {
			m.clampCur()
		}

	case msgStash:
		m.stashes = msg.v
		if m.tab == TabStash && len(m.stashes) == 0 {
			m.tab = TabStatus
		}
		m.clampCur()

	case msgRemote:
		m.remotes = msg.v
		if m.tab == TabRemote {
			m.clampCur()
		}

	case msgErr:
		m.loading = false
		m.note = &Notification{
			Message: msg.err.Error(),
			Level:   NotifyError,
			Expiry:  time.Now().Add(8 * time.Second),
		}
		cmds = append(cmds, clearNoteAfter(8*time.Second))

	case msgNotify:
		m.note = &Notification{
			Message: msg.text,
			Level:   msg.level,
			Expiry:  time.Now().Add(5 * time.Second),
		}
		d := 5 * time.Second
		if msg.level == NotifyError {
			d = 8 * time.Second
		} else if msg.level == NotifyInfo {
			d = 3 * time.Second
		}
		cmds = append(cmds, clearNoteAfter(d))
		if msg.level == NotifySuccess {
			m.loading = true
			cmds = append(cmds, doRefreshAll())
		}

	case msgClearNote:
		m.note = nil

	case msgStageAllDone:
		m.loading = false
		m.openInput(ModeCommit, "Commit message  (all files staged)", "feat: your change here…")
		cmds = append(cmds, doLoadStatus())
		return m, tea.Batch(cmds...)

	// ── Keys ──────────────────────────────────────────────────────────────────
	case tea.KeyMsg:
		// Any keypress clears the notification banner
		m.note = nil

		// Help dialog swallows all keys except those that close it
		if m.showHelp {
			if msg.Type == tea.KeyEsc || key.Matches(msg, Keys.Help) || key.Matches(msg, Keys.Quit) {
				m.showHelp = false
			}
			return m, tea.Batch(cmds...)
		}

		if m.inputMode != ModeNormal {
			nm, cmd := m.handleInputKey(msg)
			return nm, tea.Batch(append(cmds, cmd)...)
		}
		if m.confirmAction != nil {
			nm, cmd := m.handleConfirmKey(msg)
			return nm, tea.Batch(append(cmds, cmd)...)
		}
		nm, cmd := m.handleNormalKey(msg)
		return nm, tea.Batch(append(cmds, cmd)...)
	}

	return m, tea.Batch(cmds...)
}

// ── Mouse handler ─────────────────────────────────────────────────────────────

func (m Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// ── Wheel scroll (works everywhere, no action/button check needed) ────────
	if msg.Button == tea.MouseButtonWheelUp {
		if m.cur() > 0 {
			m.setCur(m.cur() - 1)
		}
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		if m.cur() < m.listLen()-1 {
			m.setCur(m.cur() + 1)
		}
		return m, nil
	}

	// Only process left-button releases for clicks
	if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	// Click anywhere dismisses notification
	if m.note != nil {
		m.note = nil
		return m, nil
	}

	// Close help dialog on click
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	switch {
	// Tab bar is now at Y=0 (no title bar above it)
	case msg.Y == 0:
		m.handleTabBarClick(msg.X)

	// Bottom bar is at Y = height-1; right side toggles help
	case msg.Y == m.height-1:
		if msg.X >= m.width-8 {
			m.showHelp = !m.showHelp
		}

	// Body starts at Y=2 (tab labels=0, tab border=1)
	default:
		const bodyStart = 2
		bH := m.bodyHeight()
		cW := m.contentW() // only clicks inside the content area (not keybind panel)
		if msg.Y >= bodyStart && msg.Y < bodyStart+bH && msg.X < cW {
			item := m.bodyClickItem(msg.Y - bodyStart)
			if item >= 0 && item < m.listLen() {
				m.setCur(item)
			}
		}
	}

	return m, nil
}

func (m *Model) handleTabBarClick(x int) {
	visible := m.visibleTabs()
	pos := 0
	for i, tab := range visible {
		meta := tabMeta[tab]
		label := meta.icon + meta.name
		var w int
		if tab == m.tab {
			w = lipglossWidth(activeTabStyle(label))
		} else {
			w = lipglossWidth(inactiveTabStyle(label))
		}
		if x >= pos && x < pos+w {
			m.tab = tab
			m.clampCur()
			return
		}
		pos += w
		if i < len(visible)-1 {
			pos += 1 // divider char
		}
	}
}

// ── Normal mode ───────────────────────────────────────────────────────────────

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, Keys.Help):
		m.showHelp = true
		return m, nil

	// Tab navigation
	case key.Matches(msg, Keys.TabNext):
		visible := m.visibleTabs()
		for i, t := range visible {
			if t == m.tab {
				m.tab = visible[(i+1)%len(visible)]
				m.clampCur()
				return m, nil
			}
		}
		if len(visible) > 0 {
			m.tab = visible[0]
			m.clampCur()
		}
		return m, nil

	case key.Matches(msg, Keys.TabPrev):
		visible := m.visibleTabs()
		for i, t := range visible {
			if t == m.tab {
				if i == 0 {
					m.tab = visible[len(visible)-1]
				} else {
					m.tab = visible[i-1]
				}
				m.clampCur()
				return m, nil
			}
		}
		return m, nil

	// Cursor movement
	case key.Matches(msg, Keys.Up):
		if m.cur() > 0 {
			m.setCur(m.cur() - 1)
		}
		return m, nil

	case key.Matches(msg, Keys.Down):
		if m.cur() < m.listLen()-1 {
			m.setCur(m.cur() + 1)
		}
		return m, nil

	case key.Matches(msg, Keys.PageUp):
		c := m.cur() - 10
		if c < 0 {
			c = 0
		}
		m.setCur(c)
		return m, nil

	case key.Matches(msg, Keys.PageDown):
		c := m.cur() + 10
		if n := m.listLen(); n > 0 && c >= n {
			c = n - 1
		}
		m.setCur(c)
		return m, nil

	case key.Matches(msg, Keys.Home):
		m.setCur(0)
		return m, nil

	case key.Matches(msg, Keys.End):
		if n := m.listLen(); n > 0 {
			m.setCur(n - 1)
		}
		return m, nil

	case key.Matches(msg, Keys.Refresh):
		m.loading = true
		return m, doRefreshAll()

	// ── Global git (any tab) ──────────────────────────────────────────────────

	case key.Matches(msg, Keys.QuickCommit):
		m.loading = true
		return m, func() tea.Msg {
			if err := git.StageAll(); err != nil {
				return msgNotify{"Stage all failed: " + err.Error(), NotifyError}
			}
			return msgStageAllDone{}
		}

	case key.Matches(msg, Keys.Push):
		return m, func() tea.Msg {
			out, err := git.Push(false)
			if err != nil {
				return msgNotify{"Push failed: " + out, NotifyError}
			}
			return msgNotify{"Pushed successfully", NotifySuccess}
		}

	case key.Matches(msg, Keys.ForcePush):
		return m, func() tea.Msg {
			out, err := git.Push(true)
			if err != nil {
				return msgNotify{"Force push failed: " + out, NotifyError}
			}
			return msgNotify{"Force pushed successfully", NotifySuccess}
		}

	case key.Matches(msg, Keys.Pull):
		return m, func() tea.Msg {
			out, err := git.Pull()
			if err != nil {
				return msgNotify{"Pull failed: " + out, NotifyError}
			}
			return msgNotify{"Pulled: " + strings.TrimSpace(out), NotifySuccess}
		}

	case key.Matches(msg, Keys.Fetch):
		return m, func() tea.Msg {
			out, err := git.Fetch()
			if err != nil {
				return msgNotify{"Fetch failed: " + out, NotifyError}
			}
			_ = out
			return msgNotify{"Fetched all remotes", NotifySuccess}
		}
	}

	switch m.tab {
	case TabStatus:
		return m.handleStatusKey(msg)
	case TabLog:
		return m.handleLogKey(msg)
	case TabBranch:
		return m.handleBranchKey(msg)
	case TabRemote:
		return m.handleRemoteKey(msg)
	case TabStash:
		return m.handleStashKey(msg)
	}
	return m, nil
}

// ── Status tab ────────────────────────────────────────────────────────────────

func (m Model) handleStatusKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Select):
		m.toggleSel(m.cur())
		if m.cur() < m.listLen()-1 {
			m.setCur(m.cur() + 1)
		}

	case key.Matches(msg, Keys.SelectAll):
		if m.selCount() == m.listLen() {
			m.clearSel()
		} else {
			for i := 0; i < m.listLen(); i++ {
				m.selected[m.tab][i] = true
			}
		}

	case key.Matches(msg, Keys.Stage):
		paths := m.selectedFilePaths()
		if len(paths) == 0 {
			if p := m.cursorFilePath(); p != "" {
				paths = []string{p}
			}
		}
		if len(paths) == 0 {
			return m, notify("Nothing to stage", NotifyInfo)
		}
		m.clearSel()
		return m, func() tea.Msg {
			for _, p := range paths {
				if err := git.StageFile(p); err != nil {
					return msgNotify{fmt.Sprintf("Stage failed: %s", err), NotifyError}
				}
			}
			return msgNotify{fmt.Sprintf("Staged %d file(s)", len(paths)), NotifySuccess}
		}

	case key.Matches(msg, Keys.StageAll):
		return m, func() tea.Msg {
			if err := git.StageAll(); err != nil {
				return msgNotify{"Stage all failed: " + err.Error(), NotifyError}
			}
			return msgNotify{"Staged all changes", NotifySuccess}
		}

	case key.Matches(msg, Keys.Unstage):
		paths := m.selectedFilePaths()
		if len(paths) == 0 {
			if p := m.cursorFilePath(); p != "" && m.isStaged(m.cur()) {
				paths = []string{p}
			}
		}
		if len(paths) == 0 {
			return m, notify("Nothing to unstage", NotifyInfo)
		}
		m.clearSel()
		return m, func() tea.Msg {
			for _, p := range paths {
				git.UnstageFile(p)
			}
			return msgNotify{fmt.Sprintf("Unstaged %d file(s)", len(paths)), NotifySuccess}
		}

	case key.Matches(msg, Keys.Commit):
		if m.status != nil && len(m.status.Staged) == 0 {
			return m, notify("No staged changes — press C to stage all & commit", NotifyInfo)
		}
		m.openInput(ModeCommit, "Commit message", "feat: describe your change…")

	case key.Matches(msg, Keys.Amend):
		m.openInput(ModeAmend, "Amend last commit (blank = keep message)", "")

	case key.Matches(msg, Keys.Stash):
		m.openInput(ModeStash, "Stash name (optional)", "WIP: work in progress…")
	}
	return m, nil
}

// ── Log tab ───────────────────────────────────────────────────────────────────

func (m Model) handleLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Select):
		m.toggleSel(m.cur())
		if m.cur() < m.listLen()-1 {
			m.setCur(m.cur() + 1)
		}

	case key.Matches(msg, Keys.SelectAll):
		if m.selCount() == m.listLen() {
			m.clearSel()
		} else {
			for i := 0; i < m.listLen(); i++ {
				m.selected[m.tab][i] = true
			}
		}

	case key.Matches(msg, Keys.Squash):
		n := m.selCount()
		if n < 2 {
			return m, notify("Select 2+ commits to squash  (space to select)", NotifyInfo)
		}
		m.openInput(ModeSquash,
			fmt.Sprintf("New message for squashing %d commits", n),
			"squash: combined commit message…")

	case key.Matches(msg, Keys.Amend):
		m.openInput(ModeAmend, "Amend last commit message (blank = keep)", "")
	}
	return m, nil
}

// ── Branch tab ────────────────────────────────────────────────────────────────

func (m Model) handleBranchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Checkout), key.Matches(msg, Keys.Enter):
		if len(m.branches) == 0 {
			return m, nil
		}
		b := m.branches[m.cur()]
		if b.IsCurrent {
			return m, notify("Already on branch: "+b.Name, NotifyInfo)
		}
		name := b.Name
		return m, func() tea.Msg {
			if err := git.CheckoutBranch(name); err != nil {
				return msgNotify{"Checkout failed: " + err.Error(), NotifyError}
			}
			return msgNotify{"Checked out: " + name, NotifySuccess}
		}

	case key.Matches(msg, Keys.NewBranch):
		m.openInput(ModeBranch, "New branch name", "feature/my-feature…")

	case key.Matches(msg, Keys.Rename):
		if len(m.branches) == 0 {
			return m, nil
		}
		b := m.branches[m.cur()]
		m.renamingBranch = b.Name
		m.openInput(ModeRename, fmt.Sprintf("Rename \"%s\" to", b.Name), b.Name)

	case key.Matches(msg, Keys.Delete):
		if len(m.branches) == 0 {
			return m, nil
		}
		b := m.branches[m.cur()]
		if b.IsCurrent {
			return m, notify("Cannot delete the current branch", NotifyError)
		}
		name := b.Name
		m.confirmMsg = fmt.Sprintf("Delete branch \"%s\"?", name)
		m.confirmAction = func() tea.Cmd {
			return func() tea.Msg {
				if err := git.DeleteBranch(name, false); err != nil {
					return msgNotify{"Delete failed: " + err.Error(), NotifyError}
				}
				return msgNotify{"Deleted branch: " + name, NotifySuccess}
			}
		}
	}
	return m, nil
}

// ── Remote tab ────────────────────────────────────────────────────────────────

func (m Model) handleRemoteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cur := m.cur()
	nActions := len(remoteActions)
	nRemotes := len(m.remotes)

	switch {
	case key.Matches(msg, Keys.Enter):
		if cur < nActions {
			return m.execRemoteAction(cur)
		} else if cur < nActions+nRemotes {
			return m, notify("e=edit URL  R=rename  d=delete", NotifyInfo)
		} else {
			m.openInput(ModeAddRemote, "Remote name", "origin")
		}

	case key.Matches(msg, Keys.NewBranch):
		if cur >= nActions {
			m.openInput(ModeAddRemote, "Remote name", "origin")
		}

	case key.Matches(msg, Keys.Edit):
		if cur >= nActions && cur < nActions+nRemotes {
			r := m.remotes[cur-nActions]
			m.renamingRemote = r.Name
			m.openInput(ModeEditRemote, fmt.Sprintf("New URL for \"%s\"", r.Name), r.FetchURL)
		}

	case key.Matches(msg, Keys.Rename):
		if cur >= nActions && cur < nActions+nRemotes {
			r := m.remotes[cur-nActions]
			m.renamingRemote = r.Name
			m.openInput(ModeRename, fmt.Sprintf("Rename remote \"%s\" to", r.Name), r.Name)
		}

	case key.Matches(msg, Keys.Delete):
		if cur >= nActions && cur < nActions+nRemotes {
			r := m.remotes[cur-nActions]
			name := r.Name
			m.confirmMsg = fmt.Sprintf("Remove remote \"%s\"?", name)
			m.confirmAction = func() tea.Cmd {
				return func() tea.Msg {
					if err := git.RemoveRemote(name); err != nil {
						return msgNotify{"Remove failed: " + err.Error(), NotifyError}
					}
					return msgNotify{"Removed remote: " + name, NotifySuccess}
				}
			}
		}
	}
	return m, nil
}

func (m Model) execRemoteAction(idx int) (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		switch idx {
		case 0:
			out, err := git.Push(false)
			if err != nil {
				return msgNotify{"Push failed: " + out, NotifyError}
			}
			return msgNotify{"Pushed successfully", NotifySuccess}
		case 1:
			out, err := git.Pull()
			if err != nil {
				return msgNotify{"Pull failed: " + out, NotifyError}
			}
			return msgNotify{"Pulled: " + strings.TrimSpace(out), NotifySuccess}
		case 2:
			out, err := git.Fetch()
			if err != nil {
				return msgNotify{"Fetch failed: " + out, NotifyError}
			}
			_ = out
			return msgNotify{"Fetched all remotes", NotifySuccess}
		case 3:
			out, err := git.Push(true)
			if err != nil {
				return msgNotify{"Force push failed: " + out, NotifyError}
			}
			return msgNotify{"Force pushed successfully", NotifySuccess}
		}
		return nil
	}
}

// ── Stash tab ─────────────────────────────────────────────────────────────────

func (m Model) handleStashKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Stash):
		m.openInput(ModeStash, "Stash name (optional)", "WIP: work in progress…")

	case key.Matches(msg, Keys.StashPop), key.Matches(msg, Keys.Enter):
		return m, func() tea.Msg {
			if err := git.StashPop(); err != nil {
				return msgNotify{"Pop failed: " + err.Error(), NotifyError}
			}
			return msgNotify{"Stash applied and removed", NotifySuccess}
		}

	case key.Matches(msg, Keys.Delete):
		idx := m.cur()
		m.confirmMsg = fmt.Sprintf("Drop stash@{%d}?", idx)
		m.confirmAction = func() tea.Cmd {
			return func() tea.Msg {
				if err := git.StashDrop(idx); err != nil {
					return msgNotify{"Drop failed: " + err.Error(), NotifyError}
				}
				return msgNotify{fmt.Sprintf("Dropped stash@{%d}", idx), NotifySuccess}
			}
		}
	}
	return m, nil
}

// ── Input overlay ─────────────────────────────────────────────────────────────

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.inputMode = ModeNormal
		m.textInput.Blur()
		return m, nil
	case tea.KeyEnter:
		return m.submitInput()
	}
	ti, cmd := m.textInput.Update(msg)
	m.textInput = ti
	return m, cmd
}

func (m Model) submitInput() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.textInput.Value())
	mode := m.inputMode
	n := m.selCount()

	m.inputMode = ModeNormal
	m.textInput.Blur()
	m.clearSel()

	switch mode {
	case ModeCommit:
		if val == "" {
			return m, notify("Commit message cannot be empty", NotifyError)
		}
		return m, func() tea.Msg {
			if err := git.CreateCommit(val); err != nil {
				return msgNotify{"Commit failed: " + err.Error(), NotifyError}
			}
			// No leading "✓" — the notification panel already adds the success icon
			return msgNotify{"Committed: " + val, NotifySuccess}
		}

	case ModeAmend:
		return m, func() tea.Msg {
			if err := git.AmendCommit(val); err != nil {
				return msgNotify{"Amend failed: " + err.Error(), NotifyError}
			}
			return msgNotify{"Commit amended", NotifySuccess}
		}

	case ModeBranch:
		if val == "" {
			return m, notify("Branch name cannot be empty", NotifyError)
		}
		return m, func() tea.Msg {
			if err := git.CreateBranch(val); err != nil {
				return msgNotify{"Create branch failed: " + err.Error(), NotifyError}
			}
			return msgNotify{"Created and switched to: " + val, NotifySuccess}
		}

	case ModeSquash:
		if val == "" {
			return m, notify("Squash message cannot be empty", NotifyError)
		}
		count := n
		return m, func() tea.Msg {
			if err := git.SquashCommits(count, val); err != nil {
				return msgNotify{"Squash failed: " + err.Error(), NotifyError}
			}
			return msgNotify{fmt.Sprintf("Squashed %d commits: %s", count, val), NotifySuccess}
		}

	case ModeStash:
		return m, func() tea.Msg {
			if err := git.Stash(val); err != nil {
				return msgNotify{"Stash failed: " + err.Error(), NotifyError}
			}
			return msgNotify{"Changes stashed", NotifySuccess}
		}

	case ModeRename:
		if val == "" {
			return m, notify("Name cannot be empty", NotifyError)
		}
		oldBranch := m.renamingBranch
		oldRemote := m.renamingRemote
		m.renamingBranch = ""
		m.renamingRemote = ""

		if oldBranch != "" {
			newName := val
			return m, func() tea.Msg {
				if err := git.RenameBranch(oldBranch, newName); err != nil {
					return msgNotify{"Rename failed: " + err.Error(), NotifyError}
				}
				return msgNotify{"Renamed: " + oldBranch + " → " + newName, NotifySuccess}
			}
		}
		if oldRemote != "" {
			newName := val
			return m, func() tea.Msg {
				if err := git.RenameRemote(oldRemote, newName); err != nil {
					return msgNotify{"Rename failed: " + err.Error(), NotifyError}
				}
				return msgNotify{"Remote renamed: " + oldRemote + " → " + newName, NotifySuccess}
			}
		}

	case ModeAddRemote:
		if val == "" {
			return m, notify("Remote name cannot be empty", NotifyError)
		}
		m.pendingRemoteName = val
		m.openInput(ModeRemoteURL, fmt.Sprintf("URL for remote \"%s\"", val), "https://github.com/user/repo.git")
		return m, nil

	case ModeRemoteURL:
		if val == "" {
			return m, notify("Remote URL cannot be empty", NotifyError)
		}
		name := m.pendingRemoteName
		m.pendingRemoteName = ""
		return m, func() tea.Msg {
			if err := git.AddRemote(name, val); err != nil {
				return msgNotify{"Add remote failed: " + err.Error(), NotifyError}
			}
			return msgNotify{"Added remote: " + name, NotifySuccess}
		}

	case ModeEditRemote:
		if val == "" {
			return m, notify("URL cannot be empty", NotifyError)
		}
		name := m.renamingRemote
		m.renamingRemote = ""
		return m, func() tea.Msg {
			if err := git.SetRemoteURL(name, val); err != nil {
				return msgNotify{"Update failed: " + err.Error(), NotifyError}
			}
			return msgNotify{"Updated URL for: " + name, NotifySuccess}
		}
	}

	return m, nil
}

// ── Confirm dialog ────────────────────────────────────────────────────────────

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y", "enter":
		action := m.confirmAction
		m.confirmAction = nil
		m.confirmMsg = ""
		return m, action()
	default:
		m.confirmAction = nil
		m.confirmMsg = ""
		return m, notify("Cancelled", NotifyInfo)
	}
}
