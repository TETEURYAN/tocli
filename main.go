package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"tocli/internal/adapter/google"
	"tocli/internal/adapter/mock"
	"tocli/internal/domain"
	"tocli/internal/ui"
	"tocli/internal/usecase"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	offline := flag.Bool("offline", false, "Use mock data only (no Google APIs)")
	syncOnly := flag.Bool("sync", false, "Validate Google auth and exit (no TUI)")
	flag.Parse()

	if *syncOnly && *offline {
		fmt.Fprintln(os.Stderr, "tocli: -sync cannot be used with -offline")
		os.Exit(1)
	}

	ctx := context.Background()

	var taskRepo domain.TaskRepository
	var eventRepo domain.EventRepository

	switch {
	case *offline:
		taskRepo = mock.NewTaskRepo()
		eventRepo = mock.NewEventRepo()
	default:
		repos, err := google.TryGoogleRepos(ctx)
		if err != nil {
			if *syncOnly {
				fmt.Fprintf(os.Stderr, "Google setup failed: %v\n", err)
				os.Exit(1)
			}
			google.WarnFallback(err)
			taskRepo = mock.NewTaskRepo()
			eventRepo = mock.NewEventRepo()
		} else {
			taskRepo, eventRepo = google.ReposAsInterfaces(repos)
			if *syncOnly {
				if _, err := taskRepo.ListTaskLists(); err != nil {
					fmt.Fprintf(os.Stderr, "Google Tasks check failed: %v\n", err)
					os.Exit(1)
				}
				now := time.Now()
				start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				if _, err := eventRepo.GetEvents(start, start.Add(24*time.Hour)); err != nil {
					fmt.Fprintf(os.Stderr, "Google Calendar check failed: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Google connection OK (Tasks + Calendar).")
				return
			}
		}
	}

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
