package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"tocli/internal/adapter/google"
	"tocli/internal/adapter/local"
	"tocli/internal/adapter/mock"
	"tocli/internal/domain"
	"tocli/internal/ui"
	"tocli/internal/usecase"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
)

var Version = "v2.0.1"

func main() {
	offline := flag.Bool("offline", false, "Use mock data only (no Google APIs)")
	syncOnly := flag.Bool("sync", false, "Validate Google auth and exit (no TUI)")
	versionFlag := flag.Bool("version", false, "Show the current version and latest available")
	updateFlag := flag.Bool("update", false, "Update tocli to the latest release from GitHub and recompile")
	themeFlag := flag.String("theme", "auto", "Color theme: auto (follow the terminal background), dark or light")
	flag.Parse()

	if *versionFlag {
		printVersion(os.Stdout)
		return
	}

	if *updateFlag {
		exe, err := os.Executable()
		if err == nil {
			exe, err = filepath.EvalSymlinks(exe)
		}
		if err == nil {
			err = runUpdate(exe, os.Stdout)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "tocli: update failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

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
		if err := google.Reachable(ctx); err != nil {
			if *syncOnly {
				fmt.Fprintf(os.Stderr, "tocli: no internet connection (required for -sync).\n%v\n", err)
				os.Exit(1)
			}
			google.WarnFallback(err)
			taskRepo = mock.NewTaskRepo()
			eventRepo = mock.NewEventRepo()
			break
		}
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

	// Daily ratings are purely local data (no Google equivalent), so they're
	// wired up independently of the Google/mock task+event repo choice above:
	// -offline gets in-memory sample data, everything else persists to disk.
	var ratingRepo domain.RatingRepository
	if *offline {
		ratingRepo = mock.NewRatingRepo()
	} else if repo, err := local.NewRatingRepo(); err != nil {
		fmt.Fprintf(os.Stderr, "tocli: could not open local ratings store (%v); daily ratings will not persist this session.\n", err)
		ratingRepo = mock.NewRatingRepo()
	} else {
		ratingRepo = repo
	}

	taskUC := usecase.NewTaskUseCase(taskRepo)
	eventUC := usecase.NewEventUseCase(eventRepo)
	contribUC := usecase.NewContributionUseCase(taskRepo)
	ratingUC := usecase.NewRatingUseCase(ratingRepo)
	progressUC := usecase.NewProgressUseCase()

	switch *themeFlag {
	case "auto", "dark", "light":
	default:
		fmt.Fprintf(os.Stderr, "tocli: unknown -theme %q (use auto, dark or light)\n", *themeFlag)
		os.Exit(2)
	}

	model := ui.NewModel(taskUC, eventUC, contribUC, ratingUC, progressUC).WithTheme(*themeFlag)

	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
