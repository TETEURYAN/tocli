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
	"os/exec"
	"net/http"
	"encoding/json"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

var Version = "v0.5.6-rc-1"

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func main() {
	offline := flag.Bool("offline", false, "Use mock data only (no Google APIs)")
	syncOnly := flag.Bool("sync", false, "Validate Google auth and exit (no TUI)")
	versionFlag := flag.Bool("version", false, "Show the current version and latest available")
	updateFlag := flag.Bool("update", false, "Update tocli to the latest release from GitHub and recompile")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("tocli %s\n", Version)
		latest, err := getLatestTag()
		if err == nil && latest != Version {
			fmt.Printf("New version available: %s\n", latest)
		} else if err == nil {
			fmt.Println("You are on the latest version.")
		}
		return
	}

	if *updateFlag {
		handleUpdate()
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

func handleUpdate() {
	fmt.Println("Checking for updates...")
	
	latestTag, err := getLatestTag()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching latest version: %v\n", err)
		os.Exit(1)
	}

	if latestTag == Version {
		fmt.Println("You already have the latest version.")
		return
	}

	fmt.Printf("Updating tocli to %s...\n", latestTag)

	// 1. Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: git is not installed or not in PATH.")
		os.Exit(1)
	}

	// 2. Git fetch and checkout
	fmt.Printf("Fetching and checking out tag %s...\n", latestTag)
	
	exec.Command("git", "fetch", "--tags").Run()
	
	cmdCheckout := exec.Command("git", "checkout", latestTag)
	cmdCheckout.Stdout = os.Stdout
	cmdCheckout.Stderr = os.Stderr
	if err := cmdCheckout.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error checking out version %s: %v\n", latestTag, err)
		// Fallback to origin/main if tag checkout fails?
		fmt.Println("Attempting to pull from main instead...")
		cmdPull := exec.Command("git", "pull", "origin", "main")
		cmdPull.Stdout = os.Stdout
		cmdPull.Stderr = os.Stderr
		if err := cmdPull.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error pulling updates: %v\n", err)
			os.Exit(1)
		}
	}

	// 3. Recompile with injected version
	fmt.Printf("Recompiling with version %s...\n", latestTag)
	
	ldFlags := fmt.Sprintf("-X main.Version=%s", latestTag)
	cmdBuild := exec.Command("go", "build", "-ldflags", ldFlags, "-o", "tocli", ".")
	cmdBuild.Stdout = os.Stdout
	cmdBuild.Stderr = os.Stderr
	if err := cmdBuild.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error recompiling: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully updated and recompiled tocli!")
}

func getLatestTag() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/TETEURYAN/tocli/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data GitHubRelease
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	return data.TagName, nil
}
