package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lonelyrower/xray-fast-deploy/internal/app"
	"github.com/lonelyrower/xray-fast-deploy/internal/tui"
)

func main() {
	a := app.New(app.Config{})
	if err := a.Run(os.Args[1:]); err != nil {
		if errors.Is(err, app.ErrTUIRequested) {
			if _, runErr := tea.NewProgram(tui.New(a.TUIData())).Run(); runErr != nil {
				fmt.Fprintln(os.Stderr, runErr)
				os.Exit(1)
			}
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
