package main

import (
	"tocli/internal/adapter/mock"
	"tocli/internal/ui"
	"tocli/internal/usecase"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	taskRepo := mock.NewTaskRepo()
	eventRepo := mock.NewEventRepo()

	taskUC := usecase.NewTaskUseCase(taskRepo)
	eventUC := usecase.NewEventUseCase(eventRepo)
	contribUC := usecase.NewContributionUseCase(taskRepo)
	progressUC := usecase.NewProgressUseCase()

	model := ui.NewModel(taskUC, eventUC, contribUC, progressUC)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
