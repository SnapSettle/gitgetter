package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all named key bindings for gitgetter.
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
	TabNext  key.Binding
	TabPrev  key.Binding

	// Selection
	Select    key.Binding
	SelectAll key.Binding

	// Git — status
	Stage    key.Binding
	Unstage  key.Binding
	StageAll key.Binding

	// Git — commits
	Commit      key.Binding
	QuickCommit key.Binding // stage all + commit from any tab
	Amend       key.Binding
	Squash      key.Binding

	// Git — remote (all work globally from any tab)
	Push      key.Binding
	ForcePush key.Binding
	Pull      key.Binding
	Fetch     key.Binding

	// Git — branches
	Checkout  key.Binding
	NewBranch key.Binding
	Delete    key.Binding
	Rename    key.Binding // rename branch or remote

	// Git — remotes management
	Edit key.Binding // edit remote URL

	// Git — stash
	Stash    key.Binding
	StashPop key.Binding

	// General
	Enter   key.Binding
	Back    key.Binding
	Refresh key.Binding
	Quit    key.Binding
	Help    key.Binding
}

// Keys is the global keymap used throughout the app.
var Keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("ctrl+u", "pgup"),
		key.WithHelp("ctrl+u", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("ctrl+d", "pgdown"),
		key.WithHelp("ctrl+d", "page down"),
	),
	Home: key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g", "top"),
	),
	End: key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G", "bottom"),
	),
	TabNext: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next tab"),
	),
	TabPrev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("⇧tab", "prev tab"),
	),

	// Selection
	Select: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle select"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "select all/none"),
	),

	// Status
	Stage: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stage"),
	),
	Unstage: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "unstage"),
	),
	StageAll: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "stage all"),
	),

	// Commits
	Commit: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "commit staged"),
	),
	QuickCommit: key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "stage all & commit"),
	),
	Amend: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "amend HEAD"),
	),
	Squash: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "squash selected"),
	),

	// Remote (global)
	Push: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "push"),
	),
	ForcePush: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "force push"),
	),
	Pull: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "pull"),
	),
	Fetch: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "fetch"),
	),

	// Branches
	Checkout: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "checkout"),
	),
	NewBranch: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Rename: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "rename"),
	),

	// Remote management
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit URL"),
	),

	// Stash
	Stash: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "stash"),
	),
	StashPop: key.NewBinding(
		key.WithKeys("T"),
		key.WithHelp("T", "pop stash"),
	),

	// General
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "confirm"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}
