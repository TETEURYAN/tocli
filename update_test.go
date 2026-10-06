package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.1.0", "v1.1.0", 0},
		{"v1.1.0", "v1.2.0", -1},
		{"v1.10.0", "v1.9.0", 1}, // numeric, not lexical
		{"v2.0.0", "v1.99.99", 1},
		{"1.0.0", "v1.0.0", 0}, // the v is optional
		{"v1.0.0-rc.1", "v1.0.0", -1},
		{"v1.0.0", "v1.0.0-rc.1", 1},
		{"v1.0.0-alpha", "v1.0.0-beta", -1},
		{"v1.0.0-rc.2", "v1.0.0-rc.10", -1},
		{"v1.0.0-rc.1", "v1.0.0-rc.1", 0},
		{"v1.0.0-1", "v1.0.0-alpha", -1}, // numeric before alphanumeric
		{"v1.0.0-rc", "v1.0.0-rc.1", -1}, // a shorter prefix is lower
		{"v1.0.0+build5", "v1.0.0", 0},   // build metadata is ignored
		{"v0.2.0-alpha", "v0.5.6-rc-1", -1},
	}
	for _, c := range cases {
		got, ok := compareVersions(c.a, c.b)
		if !ok || got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, %v; want %d", c.a, c.b, got, ok, c.want)
		}
		if rev, _ := compareVersions(c.b, c.a); rev != -c.want {
			t.Errorf("compareVersions is not antisymmetric for %q, %q", c.a, c.b)
		}
	}
	for _, bad := range []string{"", "dev", "(devel)", "1.2", "v1.x.0", "v-1.0.0"} {
		if _, ok := compareVersions(bad, "v1.0.0"); ok {
			t.Errorf("%q should not parse as a version", bad)
		}
	}
}

func TestShellWords(t *testing.T) {
	got, err := shellWords(`-s -w -X 'a/b.c=with space' -X "x.y=q\"uote" -X plain.v=1 -Xk.v=2`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-s", "-w", "-X", "a/b.c=with space", "-X", `x.y=q"uote`, "-X", "plain.v=1", "-Xk.v=2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if _, err := shellWords(`-X 'unterminated`); err == nil {
		t.Error("an unterminated quote must be an error")
	}
	if got, _ := shellWords("   "); len(got) != 0 {
		t.Errorf("blank input gave %q", got)
	}
}

func TestQuoteFlagSurvivesShellWords(t *testing.T) {
	for _, v := range []string{"main.A=plain", "main.A=with space", "main.A=it's", `main.A="dq"`} {
		words, err := shellWords("-X " + quoteFlag(v))
		if err != nil || len(words) != 2 || words[1] != v {
			t.Errorf("%q round-tripped to %q (%v)", v, words, err)
		}
	}
}

func TestCommitFromSettings(t *testing.T) {
	if got := commitFromSettings(map[string]string{}); got != "" {
		t.Errorf("no vcs info = %q", got)
	}
	if got := commitFromSettings(map[string]string{"vcs.revision": "0123456789abcdef"}); got != "0123456" {
		t.Errorf("clean = %q", got)
	}
	if got := commitFromSettings(map[string]string{"vcs.revision": "0123456789abcdef", "vcs.modified": "true"}); got != "0123456+dirty" {
		t.Errorf("dirty = %q", got)
	}
}

func TestReplaceExecutableKeepsModeAndLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	dst, next := filepath.Join(dir, "tocli"), filepath.Join(dir, "tocli.new")
	if err := os.WriteFile(dst, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(next, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceExecutable(next, dst); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dst); string(b) != "new" {
		t.Errorf("content = %q", b)
	}
	if info, _ := os.Stat(dst); info.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want the old binary's 0755", info.Mode().Perm())
	}
	for _, leftover := range []string{next, dst + ".old"} {
		if _, err := os.Stat(leftover); err == nil {
			t.Errorf("%s was left behind", leftover)
		}
	}
}

// ---- end to end: does an update really change the code, not just the number? -----------------

// fakeRelease is a git repository shaped like tocli (module "tocli", package main with a Version
// variable) with two tagged releases whose behavior differs.
type fakeRelease struct {
	repo, exe string
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "init.defaultBranch=main", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeProgram(t *testing.T, dir, feature string) {
	t.Helper()
	src := "package main\n\nimport \"fmt\"\n\nvar (\n\tVersion = \"v0.0.0\"\n\tSecret  = \"\"\n)\n\nfunc main() { fmt.Println(Version, Secret, \"" + feature + "\") }\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setupFakeRelease(t *testing.T) fakeRelease {
	t.Helper()
	for _, tool := range []string{"git", "go"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not available", tool)
		}
	}
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module tocli\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "init", "-q")

	writeProgram(t, repo, "OLD-BEHAVIOR")
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-q", "-m", "v1.0.0")
	git(t, repo, "tag", "-a", "v1.0.0", "-m", "v1.0.0")

	// Install v1.0.0 the way a user would: credentials injected at link time.
	exe := filepath.Join(root, "bin", "tocli")
	build := exec.Command("go", "build", "-ldflags", "-X main.Version=v1.0.0 -X 'main.Secret=s3cret value'", "-o", exe, ".")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the installed binary: %v\n%s", err, out)
	}

	writeProgram(t, repo, "NEW-BEHAVIOR")
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-q", "-m", "v1.1.0")
	git(t, repo, "tag", "-a", "v1.1.0", "-m", "v1.1.0")
	return fakeRelease{repo: repo, exe: exe}
}

func (f fakeRelease) run(t *testing.T) string {
	t.Helper()
	out, err := exec.Command(f.exe).CombinedOutput()
	if err != nil {
		t.Fatalf("running %s: %v\n%s", f.exe, err, out)
	}
	return strings.TrimSpace(string(out))
}

// withUpdater points the updater at the fake repository for the duration of a test.
func withUpdater(t *testing.T, f fakeRelease, installed, latest string) {
	t.Helper()
	oldURL, oldFetch, oldVersion := repoURL, fetchLatestTag, Version
	repoURL = "file://" + f.repo
	fetchLatestTag = func() (string, error) { return latest, nil }
	Version = installed
	t.Cleanup(func() { repoURL, fetchLatestTag, Version = oldURL, oldFetch, oldVersion })
}

func TestUpdateReallyChangesTheCodeNotJustTheVersion(t *testing.T) {
	f := setupFakeRelease(t)
	if got := f.run(t); got != "v1.0.0 s3cret value OLD-BEHAVIOR" {
		t.Fatalf("installed binary prints %q", got)
	}
	withUpdater(t, f, "v1.0.0", "v1.1.0")

	var out bytes.Buffer
	if err := runUpdate(f.exe, &out); err != nil {
		t.Fatalf("update failed: %v\n%s", err, out.String())
	}

	got := f.run(t)
	if !strings.Contains(got, "v1.1.0") {
		t.Errorf("the version did not change: %q", got)
	}
	if !strings.Contains(got, "NEW-BEHAVIOR") {
		t.Errorf("the version number changed but the code did not: %q", got)
	}
	if !strings.Contains(got, "s3cret value") {
		t.Errorf("the link-time credentials were dropped by the update: %q", got)
	}
	if _, err := os.Stat(f.exe + ".new"); err == nil {
		t.Error("a temporary .new binary was left next to the executable")
	}
	if !strings.Contains(out.String(), "Updated tocli to v1.1.0") {
		t.Errorf("no success message:\n%s", out.String())
	}
}

func TestUpdateEmbedsTheCommitOfTheTag(t *testing.T) {
	f := setupFakeRelease(t)
	withUpdater(t, f, "v1.0.0", "v1.1.0")
	if err := runUpdate(f.exe, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	wantSHA := git(t, f.repo, "rev-parse", "v1.1.0^{commit}")
	if err := verifyBuild(f.exe, "v1.1.0", wantSHA); err != nil {
		t.Errorf("the installed binary is not the v1.1.0 commit: %v", err)
	}
	if err := verifyBuild(f.exe, "v1.1.0", git(t, f.repo, "rev-parse", "v1.0.0^{commit}")); err == nil {
		t.Error("verifyBuild accepted a binary built from a different commit")
	}
}

func TestUpdateToAMissingTagFailsAndLeavesTheBinaryAlone(t *testing.T) {
	f := setupFakeRelease(t)
	before := f.run(t)
	withUpdater(t, f, "v1.0.0", "v9.9.9") // announced by the API but never tagged

	err := runUpdate(f.exe, &bytes.Buffer{})
	if err == nil {
		t.Fatal("updating to a tag that does not exist must fail; the old code silently fell back to main")
	}
	if got := f.run(t); got != before {
		t.Errorf("the binary changed despite the failure: %q -> %q", before, got)
	}
	if _, statErr := os.Stat(f.exe + ".new"); statErr == nil {
		t.Error("a temporary .new binary was left behind")
	}
}

func TestUpdateDoesNotTouchTheUsersCheckout(t *testing.T) {
	f := setupFakeRelease(t)
	// The user's own work: a dirty file in the repo the updater could have used.
	dirty := filepath.Join(f.repo, "notes.txt")
	if err := os.WriteFile(dirty, []byte("uncommitted"), 0o644); err != nil {
		t.Fatal(err)
	}
	headBefore := git(t, f.repo, "rev-parse", "HEAD")
	branchBefore := git(t, f.repo, "rev-parse", "--abbrev-ref", "HEAD")
	withUpdater(t, f, "v1.0.0", "v1.1.0")

	if err := runUpdate(f.exe, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if got := git(t, f.repo, "rev-parse", "HEAD"); got != headBefore {
		t.Error("the update moved HEAD in the user's repository")
	}
	if got := git(t, f.repo, "rev-parse", "--abbrev-ref", "HEAD"); got != branchBefore {
		t.Errorf("the update changed the branch to %q", got)
	}
	if b, err := os.ReadFile(dirty); err != nil || string(b) != "uncommitted" {
		t.Error("the update touched the user's uncommitted files")
	}
}

func TestUpdateWorksFromAnyDirectory(t *testing.T) {
	f := setupFakeRelease(t)
	withUpdater(t, f, "v1.0.0", "v1.1.0")
	wd, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil { // not a git repository at all
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	if err := runUpdate(f.exe, &bytes.Buffer{}); err != nil {
		t.Fatalf("update from an unrelated directory failed: %v", err)
	}
	if !strings.Contains(f.run(t), "NEW-BEHAVIOR") {
		t.Error("the code did not change")
	}
}

func TestUpdateNoOpWhenAlreadyCurrentOrNewer(t *testing.T) {
	f := setupFakeRelease(t)
	before := f.run(t)
	for _, c := range []struct{ installed, latest string }{
		{"v1.1.0", "v1.1.0"},
		{"v1.2.0", "v1.1.0"}, // a dev build ahead of the last release must not be "updated" backwards
		{"v1.1.0", "v1.1.0-rc.1"},
	} {
		withUpdater(t, f, c.installed, c.latest)
		var out bytes.Buffer
		if err := runUpdate(f.exe, &out); err != nil {
			t.Errorf("%v: %v", c, err)
		}
		if got := f.run(t); got != before {
			t.Errorf("%v: the binary changed to %q", c, got)
		}
	}
}

func TestUpdateRefusesATemporaryGoRunBinary(t *testing.T) {
	f := setupFakeRelease(t)
	withUpdater(t, f, "v1.0.0", "v1.1.0")
	err := runUpdate("/tmp/go-build123/b001/exe/tocli", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "go run") {
		t.Errorf("err = %v, want a refusal explaining go run binaries", err)
	}
}

func TestUpdateReportsAnAPIFailure(t *testing.T) {
	oldFetch := fetchLatestTag
	fetchLatestTag = func() (string, error) { return "", errors.New("offline") }
	defer func() { fetchLatestTag = oldFetch }()
	if err := runUpdate("/nonexistent/tocli", &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "offline") {
		t.Errorf("err = %v, want the API error", err)
	}
}

func TestBuildLdflagsCarriesCredentialsButNotTheOldVersion(t *testing.T) {
	f := setupFakeRelease(t)
	got, err := buildLdflags(f.exe, "v2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	words, _ := shellWords(got)
	has := func(pair string) bool {
		for i := 0; i+1 < len(words); i++ {
			if words[i] == "-X" && words[i+1] == pair {
				return true
			}
		}
		return false
	}
	if !has("main.Secret=s3cret value") {
		t.Errorf("the credential was not carried over: %s", got)
	}
	if !has("main.Version=v2.0.0") || has("main.Version=v1.0.0") {
		t.Errorf("version stamping is wrong: %s", got)
	}
	// With nothing to inherit it is just the stamp.
	if got, _ := buildLdflags(filepath.Join(t.TempDir(), "missing"), "v2.0.0"); got != "-X 'main.Version=v2.0.0'" {
		t.Errorf("no inherited flags: %q", got)
	}
}
