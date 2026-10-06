// Package platform holds the small bits of operating-system glue tocli needs.
package platform

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// OpenURL opens an http(s) URL in the user's default browser without waiting for it. Only web
// URLs are accepted: the string reaches an external program, and it can come from calendar event
// text that someone else wrote.
func OpenURL(raw string) error {
	if err := validateWebURL(raw); err != nil {
		return err
	}
	var tried []string
	for _, c := range openCommands(runtime.GOOS, isWSL(), raw) {
		path, err := exec.LookPath(c[0])
		if err != nil {
			continue
		}
		tried = append(tried, c[0])
		if err := exec.Command(path, c[1:]...).Start(); err == nil {
			return nil
		}
	}
	if len(tried) == 0 {
		return fmt.Errorf("no program found to open links on %s", runtime.GOOS)
	}
	return fmt.Errorf("could not open the browser (tried %s)", strings.Join(tried, ", "))
}

func validateWebURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("only http and https links can be opened")
	}
	return nil
}

// openCommands lists, in order of preference, the commands that can open url on a platform.
// WSL comes first for Linux under Windows: xdg-open is usually installed there but has no
// browser to launch, while wslview / explorer.exe reach the Windows default browser.
func openCommands(goos string, wsl bool, url string) [][]string {
	switch goos {
	case "linux":
		var cmds [][]string
		if wsl {
			cmds = append(cmds, []string{"wslview", url}, []string{"explorer.exe", url})
		}
		return append(cmds, []string{"xdg-open", url})
	case "darwin":
		return [][]string{{"open", url}}
	case "windows":
		return [][]string{{"rundll32", "url.dll,FileProtocolHandler", url}}
	}
	return nil
}

// isWSL reports whether this is Linux running under Windows Subsystem for Linux.
func isWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	b, err := os.ReadFile("/proc/version")
	return err == nil && strings.Contains(strings.ToLower(string(b)), "microsoft")
}
