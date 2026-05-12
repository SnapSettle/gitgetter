package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"gitgetter/internal/app"
	"gitgetter/internal/git"
)

func main() {
	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "✗  Not a git repository — run gitgetter from inside a git project.")
		os.Exit(1)
	}

	p := tea.NewProgram(
		app.InitialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gitgetter: fatal error: %v\n", err)
		os.Exit(1)
	}
}
