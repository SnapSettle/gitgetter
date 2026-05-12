package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"gitgetter/internal/git"
	"gitgetter/internal/styles"
)

// ── Shared helpers (used by both view and update) ─────────────────────────────

func activeTabStyle(label string) string   { return styles.ActiveTab.Render(label) }
func inactiveTabStyle(label string) string { return styles.Tab.Render(label) }
func lipglossWidth(s string) int           { return lipgloss.Width(s) }

func wordWrap(s string, width int) string {
	if width <= 4 {
		return s
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	var lines []string
	cur := ""
	for _, w := range words {
		if cur == "" {
			cur = w
		} else if len([]rune(cur))+1+len([]rune(w)) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return strings.Join(lines, "\n")
}

// ── Keybind data ──────────────────────────────────────────────────────────────

// tabSpecificBindings returns context-sensitive bindings for the current tab.
func (m Model) tabSpecificBindings() [][2]string {
	switch m.tab {
	case TabStatus:
		return [][2]string{
			{"space", "select"},
			{"s", "stage"},
			{"u", "unstage"},
			{"S", "stage all"},
			{"c", "commit"},
			{"A", "amend"},
			{"t", "stash"},
		}
	case TabLog:
		return [][2]string{
			{"space", "select"},
			{"x", "squash"},
			{"A", "amend"},
		}
	case TabBranch:
		return [][2]string{
			{"↵/o", "checkout"},
			{"n", "new"},
			{"R", "rename"},
			{"d", "delete"},
		}
	case TabRemote:
		return [][2]string{
			{"↵", "run action"},
			{"n", "add remote"},
			{"e", "edit URL"},
			{"R", "rename"},
			{"d", "delete"},
		}
	case TabStash:
		return [][2]string{
			{"t", "stash"},
			{"↵/T", "pop"},
			{"d", "drop"},
		}
	}
	return nil
}

// globalGitBindings are always-available git operations.
func (m Model) globalGitBindings() [][2]string {
	return [][2]string{
		{"C", "commit all"},
		{"p", "push"},
		{"P", "force push"},
		{"l", "pull"},
		{"f", "fetch"},
	}
}

// navBindings are always-available navigation keys.
func navBindings() [][2]string {
	return [][2]string{
		{"↑↓/jk", "move"},
		{"g/G", "top/bot"},
		{"ctrl+u/d", "page"},
		{"tab/⇧tab", "switch"},
		{"scroll", "scroll"},
		{"r", "refresh"},
		{"?", "help"},
		{"q", "quit"},
	}
}

// ── Entry point ───────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	// Full-screen modals via lipgloss.Place (no ANSI string splicing)
	if m.showHelp {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.viewHelpDialog())
	}
	if m.inputMode != ModeNormal {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.viewInputDialog())
	}
	if m.confirmAction != nil {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.viewConfirmDialog())
	}

	// Normal view — title bar is now at the BOTTOM
	var rows []string
	rows = append(rows, m.viewTabBar()) // Y=0: tab labels (mouse-clickable)
	rows = append(rows, m.viewBody())   // Y=2+: body + keybind panel on right
	if m.note != nil {
		rows = append(rows, m.viewNotifyPanel()) // notification replaces bottom bar
	} else {
		rows = append(rows, m.viewBottomBar()) // Y=H-1: branch/status + [?] button
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// ── Tab bar (Y=0, mouse-clickable) ───────────────────────────────────────────

func (m Model) viewTabBar() string {
	visible := m.visibleTabs()
	var parts []string
	for i, tab := range visible {
		meta := tabMeta[tab]
		label := meta.icon + meta.name
		if tab == m.tab {
			parts = append(parts, styles.ActiveTab.Render(label))
		} else {
			parts = append(parts, styles.Tab.Render(label))
		}
		if i < len(visible)-1 {
			parts = append(parts, styles.TabDivider.Render("│"))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Center, parts...)
	pad := m.width - lipgloss.Width(bar)
	if pad > 0 {
		bar += strings.Repeat(" ", pad)
	}
	return lipgloss.NewStyle().
		Background(styles.ColorBgAlt).
		BorderBottom(true).
		BorderForeground(styles.ColorBorder).
		Render(bar)
}

// ── Bottom bar (moved from top) — branch info + [?] button ───────────────────

func (m Model) viewBottomBar() string {
	logo := styles.TitleLogo.Render(" gitgetter")

	info := ""
	if m.status != nil {
		info = "  " + styles.BranchCurrent.Render(" "+m.status.Branch)
		if m.status.Upstream != "" {
			info += styles.Muted.Render("  " + m.status.Upstream)
		}
		if m.status.Ahead > 0 {
			info += "  " + styles.PillGreen.Render(fmt.Sprintf("↑%d", m.status.Ahead))
		}
		if m.status.Behind > 0 {
			info += " " + styles.PillYellow.Render(fmt.Sprintf("↓%d", m.status.Behind))
		}
	}
	if m.loading {
		info += "  " + m.spinner.View()
	}

	left := logo + info

	// Help button — right-aligned, click target for mouse (X >= width-8)
	helpBtn := "  " + styles.HelpKey.Render("?") + " " + styles.HelpDesc.Render("help") + " "

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(helpBtn)
	if gap < 1 {
		gap = 1
	}

	return styles.TitleBar.Width(m.width).Render(left + strings.Repeat(" ", gap) + helpBtn)
}

// ── Body (content + optional right-side keybind panel) ───────────────────────

func (m Model) viewBody() string {
	h := m.bodyHeight()
	kW := m.keybindPanelW()
	cW := m.contentW()

	var content string
	switch m.tab {
	case TabStatus:
		content = m.viewStatus(h, cW)
	case TabLog:
		content = m.viewLog(h, cW)
	case TabBranch:
		content = m.viewBranches(h, cW)
	case TabRemote:
		content = m.viewRemote(h, cW)
	case TabStash:
		content = m.viewStash(h, cW)
	}

	mainPane := lipgloss.NewStyle().Width(cW).Height(h).Render(content)

	if kW > 0 {
		kbPane := m.viewKeybindPanel(kW, h)
		return lipgloss.JoinHorizontal(lipgloss.Top, mainPane, kbPane)
	}
	return mainPane
}

// ── Right-side keybind panel ──────────────────────────────────────────────────

func (m Model) viewKeybindPanel(w, h int) string {
	// w = total panel width including border
	// inner content width = w - border(1) - padding(1) = w-2
	inner := w - 2
	keyW := 6 // fixed width for key column

	entry := func(k, d string) string {
		key := styles.HelpKey.Render(fmt.Sprintf("%-*s", keyW, k))
		desc := styles.HelpDesc.Render(styles.Truncate(d, inner-keyW-1))
		return key + " " + desc
	}

	var lines []string

	// ── Context section (tab-specific) ───────────────────────────────────────
	ctx := m.tabSpecificBindings()
	if len(ctx) > 0 {
		tabName := strings.ToUpper(tabMeta[m.tab].name)
		lines = append(lines, styles.SectionTitle.Render(tabName))
		for _, b := range ctx {
			lines = append(lines, entry(b[0], b[1]))
		}
		lines = append(lines, "")
	}

	// ── Global git section ────────────────────────────────────────────────────
	lines = append(lines, styles.SectionTitle.Render("GLOBAL"))
	for _, b := range m.globalGitBindings() {
		lines = append(lines, entry(b[0], b[1]))
	}
	lines = append(lines, "")

	// ── Navigation (at the bottom as requested) ───────────────────────────────
	lines = append(lines, styles.SectionTitle.Render("NAV"))
	for _, b := range navBindings() {
		lines = append(lines, entry(b[0], b[1]))
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorBorder).
		Width(inner).
		Height(h).
		PaddingLeft(1).
		Render(content)
}

// ── Help dialog ───────────────────────────────────────────────────────────────

func (m Model) viewHelpDialog() string {
	dialogW := m.width - 6
	if dialogW > 84 {
		dialogW = 84
	}
	if dialogW < 44 {
		dialogW = 44
	}

	// Inner width (border 1 each side + padding 1 each side = -4)
	inner := dialogW - 4
	twoCol := inner >= 60
	colW := 0
	if twoCol {
		colW = (inner - 2) / 2 // -2 for the column divider gap
	}

	// Section builder
	buildSection := func(title string, binds [][2]string) string {
		var sb strings.Builder
		sb.WriteString(styles.SectionTitle.Render(title) + "\n")
		for _, b := range binds {
			sb.WriteString(fmt.Sprintf("  %-8s %s\n", b[0], b[1]))
		}
		return sb.String()
	}

	// Left column content
	leftContent := buildSection("STATUS", [][2]string{
		{"space", "toggle select"},
		{"s", "stage file(s)"},
		{"u", "unstage file(s)"},
		{"S", "stage all changes"},
		{"c", "commit staged"},
		{"A", "amend last commit"},
		{"t", "stash changes"},
	}) + "\n" + buildSection("LOG", [][2]string{
		{"space", "toggle select"},
		{"x", "squash selected"},
		{"A", "amend HEAD"},
	}) + "\n" + buildSection("GLOBAL", [][2]string{
		{"C", "stage all & commit"},
		{"p", "push"},
		{"P", "force push (--lease)"},
		{"l", "pull"},
		{"f", "fetch all remotes"},
		{"r", "refresh"},
		{"q", "quit"},
	})

	// Right column content
	rightContent := buildSection("BRANCHES", [][2]string{
		{"↵/o", "checkout branch"},
		{"n", "new branch"},
		{"R", "rename branch"},
		{"d", "delete branch"},
	}) + "\n" + buildSection("REMOTE", [][2]string{
		{"↵", "run selected action"},
		{"n", "add new remote"},
		{"e", "edit remote URL"},
		{"R", "rename remote"},
		{"d", "delete remote"},
	}) + "\n" + buildSection("STASH", [][2]string{
		{"t", "stash changes"},
		{"↵/T", "pop stash"},
		{"d", "drop stash"},
	}) + "\n" + buildSection("NAVIGATION", [][2]string{
		{"↑↓/jk", "move cursor"},
		{"ctrl+u/d", "page up/down"},
		{"g/G", "top/bottom"},
		{"tab/⇧tab", "switch tabs"},
		{"scroll", "scroll list"},
		{"click tab", "switch tab"},
		{"click item", "move cursor"},
		{"?", "toggle this help"},
	})

	var body string
	if twoCol {
		left := lipgloss.NewStyle().Width(colW).Render(leftContent)
		// right := lipgloss.NewStyle().Width(colW).Render(rightContent)
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			left,
			lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(styles.ColorBorder).
				PaddingLeft(1).
				Width(colW).
				Render(rightContent),
		)
	} else {
		body = leftContent + "\n" + rightContent
	}

	title := styles.TitleLogo.Render(" gitgetter ") +
		"  " + styles.Bold.Render("keyboard reference")

	footer := styles.HelpDesc.Render("esc · ? · q   close")

	full := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		body,
		"",
		styles.Muted.Render(strings.Repeat("─", inner)),
		footer,
	)

	return styles.FocusedPanel.Width(dialogW - 4).Render(full)
}

// ── Status tab ────────────────────────────────────────────────────────────────

func (m Model) viewStatus(h, w int) string {
	if m.status == nil {
		return m.viewLoading(h)
	}

	var sb strings.Builder
	idx := 0

	section := func(title string, files []git.FileStatus, iconFn func(string) string) {
		if len(files) == 0 {
			return
		}
		sb.WriteString(styles.SectionTitle.Render(title) + "\n")
		for _, f := range files {
			cursor := "  "
			if idx == m.cur() {
				cursor = styles.ItemCursor.Render(" ❯")
			}
			check := "  "
			if m.isSel(idx) {
				check = styles.ItemChecked.Render(" ✔")
			}
			icon := iconFn(f.StagedStatus + f.UnstagedStatus)
			path := styles.Truncate(f.Path, w-20)

			var row string
			if idx == m.cur() {
				row = styles.ItemSelected.Width(w - 2).Render(
					cursor + check + " " + icon + " " + path)
			} else {
				row = cursor + check + " " + icon + " " + styles.ItemNormal.Render(path)
			}
			sb.WriteString(row + "\n")
			idx++
		}
		sb.WriteString("\n")
	}

	stagedIcon := func(status string) string {
		switch status {
		case "deleted":
			return styles.IconDeleted.Render("D")
		case "renamed":
			return styles.IconRenamed.Render("R")
		default:
			return styles.IconStaged.Render("A")
		}
	}
	unstagedIcon := func(status string) string {
		if status == "deleted" {
			return styles.IconDeleted.Render("D")
		}
		return styles.IconUnstaged.Render("M")
	}
	untrackedIcon := func(_ string) string { return styles.IconUntracked.Render("?") }

	section("  Staged", m.status.Staged, stagedIcon)
	section("  Modified", m.status.Unstaged, unstagedIcon)
	section("  Untracked", m.status.Untracked, untrackedIcon)

	if m.status.IsClean {
		sb.WriteString("\n  " + styles.PillGreen.Render(" Clean ") +
			styles.Muted.Render("  nothing to commit, working tree clean") + "\n")
	}

	return scrollView(sb.String(), m.cur(), h)
}

// ── Log tab ───────────────────────────────────────────────────────────────────

func (m Model) viewLog(h, w int) string {
	if len(m.commits) == 0 {
		if m.loading {
			return m.viewLoading(h)
		}
		return styles.Muted.Render("\n  No commits found")
	}

	hashW := 8
	metaW := 26
	msgW := w - hashW - metaW - 10

	var sb strings.Builder
	for i, c := range m.commits {
		cursor := "  "
		if i == m.cur() {
			cursor = styles.ItemCursor.Render(" ❯")
		}
		check := "  "
		if m.isSel(i) {
			check = styles.ItemChecked.Render(" ✔")
		}

		hash := styles.CommitHash.Render(fmt.Sprintf("%-*s", hashW, c.Short))
		msg := styles.Truncate(c.Message, msgW)
		meta := styles.CommitMeta.Render(styles.Truncate(c.Author+" · "+c.Date, metaW))

		refs := ""
		if c.Refs != "" {
			for _, ref := range strings.Split(c.Refs, ", ") {
				ref = strings.TrimSpace(ref)
				if ref == "" {
					continue
				}
				if strings.HasPrefix(ref, "HEAD") {
					refs += " " + styles.RefHead.Render(ref)
				} else if strings.HasPrefix(ref, "tag:") {
					refs += " " + styles.RefTag.Render(strings.TrimPrefix(ref, "tag: "))
				} else {
					refs += " " + styles.RefBranch.Render(ref)
				}
			}
		}

		var line string
		if i == m.cur() {
			line = styles.ItemSelected.Width(w-2).Render(
				cursor+check+" "+hash+"  "+msg+refs) +
				"\n        " + meta + "\n"
		} else {
			line = cursor + check + " " + hash + "  " + styles.CommitMsg.Render(msg) + refs + "\n" +
				"        " + meta + "\n"
		}
		sb.WriteString(line)
	}

	return scrollView(sb.String(), m.cur()*2, h)
}

// ── Branches tab ──────────────────────────────────────────────────────────────

func (m Model) viewBranches(h, w int) string {
	if len(m.branches) == 0 {
		if m.loading {
			return m.viewLoading(h)
		}
		return styles.Muted.Render("\n  No branches found")
	}

	var sb strings.Builder
	for i, b := range m.branches {
		cursor := "  "
		if i == m.cur() {
			cursor = styles.ItemCursor.Render(" ❯")
		}

		icon := " "
		nameStyle := styles.BranchLocal
		if b.IsCurrent {
			icon = " "
			nameStyle = styles.BranchCurrent
		} else if b.IsRemote {
			icon = " "
			nameStyle = styles.BranchRemote
		}

		upstream := ""
		if b.Upstream != "" {
			upstream = styles.Muted.Render("  → " + b.Upstream)
		}
		current := ""
		if b.IsCurrent {
			current = "  " + styles.PillGreen.Render(" current ")
		}

		var line string
		if i == m.cur() {
			line = styles.ItemSelected.Width(w-2).
				Render(cursor+" "+icon+" "+b.Name+upstream+strings.TrimSpace(current)) + "\n"
		} else {
			line = cursor + " " + icon + " " + nameStyle.Render(b.Name) + upstream + current + "\n"
		}
		sb.WriteString(line)
	}

	return scrollView(sb.String(), m.cur(), h)
}

// ── Remote tab — order: Actions → Remotes → Sync (compact, 2 lines) ──────────

func (m Model) viewRemote(h, w int) string {
	var sb strings.Builder
	nActions := len(remoteActions)

	// ── 1. Actions section ────────────────────────────────────────────────────
	sb.WriteString(styles.SectionTitle.Render("  Actions") + "\n")
	for i, a := range remoteActions {
		cursor := "  "
		if i == m.cur() {
			cursor = styles.ItemCursor.Render(" ❯")
		}
		if i == m.cur() {
			sb.WriteString(styles.ItemSelected.Width(w-2).
				Render(cursor+" "+a.icon+" "+a.label+"  "+styles.Muted.Render(a.desc)) + "\n")
		} else {
			sb.WriteString(cursor + " " + a.icon + " " + styles.Bold.Render(a.label) +
				styles.Muted.Render("  "+a.desc) + "\n")
		}
	}

	// ── 2. Remotes section ────────────────────────────────────────────────────
	sb.WriteString("\n")
	sb.WriteString(styles.SectionTitle.Render("  Remotes") + "\n")

	if len(m.remotes) == 0 && !m.loading {
		sb.WriteString("  " + styles.Muted.Render("no remotes configured") + "\n")
	}

	for i, r := range m.remotes {
		idx := nActions + i
		cursor := "  "
		if idx == m.cur() {
			cursor = styles.ItemCursor.Render(" ❯")
		}
		url := r.FetchURL
		if url == "" {
			url = r.PushURL
		}
		urlTrunc := styles.Truncate(url, w-len(r.Name)-14)
		if idx == m.cur() {
			sb.WriteString(styles.ItemSelected.Width(w-2).
				Render(cursor+"  "+r.Name+"  "+urlTrunc) + "\n")
		} else {
			sb.WriteString(cursor + "  " + styles.BranchCurrent.Render(r.Name) +
				"  " + styles.Muted.Render(urlTrunc) + "\n")
		}
	}

	// "Add remote" row
	addIdx := nActions + len(m.remotes)
	addCursor := "  "
	if addIdx == m.cur() {
		addCursor = styles.ItemCursor.Render(" ❯")
	}
	if addIdx == m.cur() {
		sb.WriteString(styles.ItemSelected.Width(w-2).Render(addCursor+" + Add remote") + "\n")
	} else {
		sb.WriteString(addCursor + " " + styles.Muted.Render("+ Add remote") + "\n")
	}

	// ── 3. Sync section — compact 2 lines at the bottom ──────────────────────
	if m.status != nil && m.status.Upstream != "" {
		sb.WriteString("\n")
		sb.WriteString(styles.SectionTitle.Render("  Sync") + "\n")

		// Single line: upstream name + ahead/behind status
		upstreamStr := styles.BranchRemote.Render(m.status.Upstream)
		var statusStr string
		if m.status.Ahead == 0 && m.status.Behind == 0 {
			statusStr = styles.PillGray.Render(" in sync ")
		} else {
			var parts []string
			if m.status.Ahead > 0 {
				parts = append(parts, styles.PillGreen.Render(fmt.Sprintf(" ↑%d ", m.status.Ahead)))
			}
			if m.status.Behind > 0 {
				parts = append(parts, styles.PillYellow.Render(fmt.Sprintf(" ↓%d ", m.status.Behind)))
			}
			statusStr = strings.Join(parts, " ")
		}
		sb.WriteString("  " + upstreamStr + "   " + statusStr + "\n")
	}

	return lipgloss.NewStyle().Width(w).Render(sb.String())
}

// ── Stash tab ─────────────────────────────────────────────────────────────────

func (m Model) viewStash(h, w int) string {
	var sb strings.Builder
	sb.WriteString(styles.SectionTitle.Render("  Stashes") + "\n\n")

	if len(m.stashes) == 0 {
		sb.WriteString("  " + styles.PillGray.Render(" empty ") +
			styles.Muted.Render("  press t to stash current changes") + "\n")
		return scrollView(sb.String(), 0, h)
	}

	for i, s := range m.stashes {
		cursor := "  "
		if i == m.cur() {
			cursor = styles.ItemCursor.Render(" ❯")
		}
		label := styles.Truncate(s, w-10)
		if i == m.cur() {
			sb.WriteString(styles.ItemSelected.Width(w-2).
				Render(cursor+" 󰏨 "+label) + "\n")
		} else {
			sb.WriteString(cursor + " 󰏨 " + styles.ItemNormal.Render(label) + "\n")
		}
	}

	return scrollView(sb.String(), m.cur(), h)
}

// ── Notification panel (replaces bottom bar when active) ──────────────────────

func (m Model) viewNotifyPanel() string {
	if m.note == nil {
		return ""
	}

	var icon string
	var msgStyle lipgloss.Style

	switch m.note.Level {
	case NotifySuccess:
		icon = "✓"
		msgStyle = styles.NotifSuccess
	case NotifyError:
		icon = "✗"
		msgStyle = styles.NotifError
	default:
		icon = "ℹ"
		msgStyle = styles.NotifInfo
	}

	msg := styles.Truncate(m.note.Message, m.width-8)
	line1 := msgStyle.Width(m.width).Render(" " + icon + "  " + msg)
	line2 := styles.HelpBar.Width(m.width).Render("  press any key to dismiss")
	return line1 + "\n" + line2
}

// ── Input dialog ──────────────────────────────────────────────────────────────

func (m Model) viewInputDialog() string {
	label := styles.InputLabel.Render(m.inputLabel)
	hint := styles.Muted.Render(m.inputHint)
	input := m.textInput.View()

	body := lipgloss.JoinVertical(lipgloss.Left,
		label, hint, "", input, "",
		styles.HelpDesc.Render("↵ confirm  ·  esc cancel"),
	)
	return styles.FocusedPanel.Width(68).Render(body)
}

// ── Confirm dialog ────────────────────────────────────────────────────────────

func (m Model) viewConfirmDialog() string {
	title := styles.ConfirmTitle.Render("⚠  Confirm")
	msg := styles.Base.Render(m.confirmMsg)
	actions := styles.HelpDesc.Render("y / ↵  yes  ·  any other key  cancel")
	body := lipgloss.JoinVertical(lipgloss.Left, title, "", msg, "", actions)
	return styles.ConfirmBox.Width(52).Render(body)
}

// ── Loading placeholder ───────────────────────────────────────────────────────

func (m Model) viewLoading(h int) string {
	line := "  " + m.spinner.View() + " " + styles.Muted.Render("Loading git data…")
	return lipgloss.NewStyle().Height(h).Render(line)
}

// ── scrollView ────────────────────────────────────────────────────────────────

func scrollView(content string, cursorLine, h int) string {
	lines := strings.Split(content, "\n")
	total := len(lines)
	if total <= h {
		return content
	}
	start := cursorLine - h/2
	if start < 0 {
		start = 0
	}
	end := start + h
	if end > total {
		end = total
		start = end - h
		if start < 0 {
			start = 0
		}
	}
	return strings.Join(lines[start:end], "\n")
}
