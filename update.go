package main

import (
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// Overridable so tests can point the updater at a local repository.
var (
	repoURL        = "https://github.com/TETEURYAN/tocli.git"
	releasesURL    = "https://api.github.com/repos/TETEURYAN/tocli/releases/latest"
	fetchLatestTag = getLatestTag
	httpClient     = &http.Client{Timeout: 10 * time.Second}
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func getLatestTag() (string, error) {
	req, err := http.NewRequest(http.MethodGet, releasesURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "tocli-updater")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
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
	if data.TagName == "" {
		return "", errors.New("GitHub API returned a release without a tag")
	}
	return data.TagName, nil
}

// buildCommit is the git revision this binary was built from (short), with "+dirty" when the
// working tree had uncommitted changes. Empty when the build carries no VCS information.
func buildCommit() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return commitFromSettings(settingsMap(bi.Settings))
}

func settingsMap(settings []debug.BuildSetting) map[string]string {
	m := make(map[string]string, len(settings))
	for _, s := range settings {
		m[s.Key] = s.Value
	}
	return m
}

func commitFromSettings(s map[string]string) string {
	rev := s["vcs.revision"]
	if rev == "" {
		return ""
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if s["vcs.modified"] == "true" {
		rev += "+dirty"
	}
	return rev
}

// versionLine is what -version prints first: the version number plus the commit it was built
// from, so "which code is this?" has an answer that is not just a label.
func versionLine() string {
	if c := buildCommit(); c != "" {
		return fmt.Sprintf("tocli %s (commit %s)", Version, c)
	}
	return fmt.Sprintf("tocli %s", Version)
}

// printVersion implements -version.
func printVersion(w io.Writer) {
	fmt.Fprintln(w, versionLine())
	latest, err := fetchLatestTag()
	if err != nil {
		return
	}
	switch cmp, ok := compareVersions(Version, latest); {
	case ok && cmp < 0, !ok && latest != Version:
		fmt.Fprintf(w, "New version available: %s\n", latest)
	case ok && cmp > 0:
		fmt.Fprintf(w, "You are running a newer build than the latest release (%s).\n", latest)
	default:
		fmt.Fprintln(w, "You are on the latest version.")
	}
}

// runUpdate replaces the executable at exe with the latest release. It never touches the user's
// checkout: the release tag is cloned into a temporary directory, built there, checked, and only
// then swapped in. Any failure leaves the current binary exactly as it was.
func runUpdate(exe string, out io.Writer) error {
	fmt.Fprintln(out, "Checking for updates...")
	latest, err := fetchLatestTag()
	if err != nil {
		return fmt.Errorf("fetching latest version: %w", err)
	}

	if cmp, ok := compareVersions(Version, latest); ok {
		switch {
		case cmp == 0:
			fmt.Fprintln(out, "You already have the latest version.")
			return nil
		case cmp > 0:
			fmt.Fprintf(out, "Your version (%s) is newer than the latest release (%s); nothing to do.\n", Version, latest)
			return nil
		}
	} else if latest == Version {
		fmt.Fprintln(out, "You already have the latest version.")
		return nil
	}

	for _, tool := range []string{"git", "go"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s is required to update (it builds from source) but is not in PATH", tool)
		}
	}
	if strings.Contains(filepath.ToSlash(exe), "/go-build") {
		return errors.New("this is a temporary `go run` binary; build and run the installed tocli to update it")
	}

	fmt.Fprintf(out, "Updating tocli %s -> %s\n", Version, latest)

	tmp, err := os.MkdirTemp("", "tocli-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	src := filepath.Join(tmp, "src")

	fmt.Fprintf(out, "Fetching %s...\n", latest)
	// --branch accepts a tag and fails if it does not exist; there is deliberately no fallback to
	// another branch, which would build different code under the new version number.
	if err := runCmd(out, tmp, "git", "clone", "--quiet", "--depth", "1", "--branch", latest, repoURL, src); err != nil {
		return fmt.Errorf("could not fetch release %s: %w", latest, err)
	}
	sha, err := outputCmd(src, "git", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("reading the release commit: %w", err)
	}

	ldflags, err := buildLdflags(exe, latest)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Building %s (commit %s)...\n", latest, short(sha))
	next := exe + ".new"
	defer os.Remove(next) // gone already after a successful swap
	if err := runCmd(out, src, "go", "build", "-ldflags", ldflags, "-o", next, "."); err != nil {
		return fmt.Errorf("building %s (the current binary was not changed): %w", latest, err)
	}

	if err := verifyBuild(next, latest, sha); err != nil {
		return fmt.Errorf("the new binary failed verification, keeping the current one: %w", err)
	}
	if err := replaceExecutable(next, exe); err != nil {
		return fmt.Errorf("installing the new binary at %s: %w (try running with sudo, or move tocli somewhere writable)", exe, err)
	}

	fmt.Fprintf(out, "Updated tocli to %s (commit %s). Restart tocli to use it.\n", latest, short(sha))
	return nil
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func runCmd(out io.Writer, dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func outputCmd(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	b, err := cmd.Output()
	return strings.TrimSpace(string(b)), err
}

// verifyBuild reads the build information embedded in the freshly built binary and checks that
// it really is the release: built from the tag's commit, from a clean tree, stamped with the tag.
func verifyBuild(path, tag, wantSHA string) error {
	bi, err := buildinfo.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading build info: %w", err)
	}
	s := settingsMap(bi.Settings)
	if got := s["vcs.revision"]; got != wantSHA {
		return fmt.Errorf("built from commit %q, expected %s", got, short(wantSHA))
	}
	if s["vcs.modified"] == "true" {
		return errors.New("built from a modified source tree")
	}
	if !strings.Contains(s["-ldflags"], "main.Version="+tag) {
		return fmt.Errorf("binary is not stamped with %s", tag)
	}
	return nil
}

// buildLdflags is the -ldflags for the new build: the version stamp, plus every other -X
// assignment (and -s/-w) the running binary was built with. Without this an update would drop the
// embedded Google OAuth credentials, because they only exist as link-time values.
func buildLdflags(exe, tag string) (string, error) {
	var parts []string
	if bi, err := buildinfo.ReadFile(exe); err == nil {
		prev := settingsMap(bi.Settings)["-ldflags"]
		words, err := shellWords(prev)
		if err != nil {
			return "", fmt.Errorf("reading the current binary's link flags: %w", err)
		}
		for i := 0; i < len(words); i++ {
			w := words[i]
			switch {
			case w == "-s" || w == "-w":
				parts = append(parts, w)
			case w == "-X" && i+1 < len(words):
				i++
				if !strings.HasPrefix(words[i], "main.Version=") {
					parts = append(parts, "-X "+quoteFlag(words[i]))
				}
			case strings.HasPrefix(w, "-X") && len(w) > 2:
				if v := w[2:]; !strings.HasPrefix(v, "main.Version=") {
					parts = append(parts, "-X "+quoteFlag(v))
				}
			}
		}
	}
	parts = append(parts, "-X "+quoteFlag("main.Version="+tag))
	return strings.Join(parts, " "), nil
}

func quoteFlag(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// shellWords splits s like a POSIX shell: whitespace separates words, single and double quotes
// group, backslash escapes outside single quotes. The go tool records -ldflags that way.
func shellWords(s string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord := false
	var quote rune
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case quote == '"':
			switch {
			case r == '"':
				quote = 0
			case r == '\\' && i+1 < len(rs):
				i++
				cur.WriteRune(rs[i])
			default:
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case r == '\\' && i+1 < len(rs):
			i++
			cur.WriteRune(rs[i])
			inWord = true
		case r == ' ' || r == '\t' || r == '\n':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote")
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}

// replaceExecutable swaps next in for dst, keeping dst's permissions. A rename over a running
// binary is fine on Unix; where it is refused (Windows) the old file is moved aside first.
func replaceExecutable(next, dst string) error {
	if info, err := os.Stat(dst); err == nil {
		if err := os.Chmod(next, info.Mode().Perm()); err != nil {
			return err
		}
	}
	if err := os.Rename(next, dst); err == nil {
		return nil
	}
	old := dst + ".old"
	_ = os.Remove(old)
	if err := os.Rename(dst, old); err != nil {
		return err
	}
	if err := os.Rename(next, dst); err != nil {
		_ = os.Rename(old, dst) // put the original back
		return err
	}
	_ = os.Remove(old)
	return nil
}

// ---- versions --------------------------------------------------------------------------------

type semver struct {
	major, minor, patch int
	pre                 []string
}

// parseVersion reads vMAJOR.MINOR.PATCH[-prerelease][+build]; the leading v is optional.
func parseVersion(s string) (semver, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var v semver
	if i := strings.IndexByte(s, '-'); i >= 0 {
		v.pre = strings.Split(s[i+1:], ".")
		s = s[:i]
	}
	nums := strings.Split(s, ".")
	if len(nums) != 3 {
		return semver{}, false
	}
	for i, p := range []*int{&v.major, &v.minor, &v.patch} {
		n, err := strconv.Atoi(nums[i])
		if err != nil || n < 0 {
			return semver{}, false
		}
		*p = n
	}
	return v, true
}

// compareVersions returns -1, 0 or 1 following SemVer precedence; ok is false when either side
// is not a version number (a development build, say).
func compareVersions(a, b string) (cmp int, ok bool) {
	va, okA := parseVersion(a)
	vb, okB := parseVersion(b)
	if !okA || !okB {
		return 0, false
	}
	for _, d := range [][2]int{{va.major, vb.major}, {va.minor, vb.minor}, {va.patch, vb.patch}} {
		if d[0] != d[1] {
			if d[0] < d[1] {
				return -1, true
			}
			return 1, true
		}
	}
	switch {
	case len(va.pre) == 0 && len(vb.pre) == 0:
		return 0, true
	case len(va.pre) == 0:
		return 1, true // 1.0.0 > 1.0.0-rc.1
	case len(vb.pre) == 0:
		return -1, true
	}
	for i := 0; i < len(va.pre) && i < len(vb.pre); i++ {
		if c := comparePre(va.pre[i], vb.pre[i]); c != 0 {
			return c, true
		}
	}
	switch {
	case len(va.pre) < len(vb.pre):
		return -1, true
	case len(va.pre) > len(vb.pre):
		return 1, true
	}
	return 0, true
}

func comparePre(a, b string) int {
	na, errA := strconv.Atoi(a)
	nb, errB := strconv.Atoi(b)
	switch {
	case errA == nil && errB == nil:
		if na < nb {
			return -1
		} else if na > nb {
			return 1
		}
		return 0
	case errA == nil:
		return -1 // numeric identifiers sort before alphanumeric ones
	case errB == nil:
		return 1
	}
	return strings.Compare(a, b)
}
