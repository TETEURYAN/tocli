package platform

import (
	"reflect"
	"testing"
)

func TestValidateWebURL(t *testing.T) {
	for _, ok := range []string{"https://meet.google.com/abc-defg-hij", "http://example.com", "  https://a.b/c?d=e&f=g  "} {
		if err := validateWebURL(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{
		"", "meet.google.com", "file:///etc/passwd", "javascript:alert(1)", "ftp://x.y",
		"https://", "-flag", "https:///nohost", "mailto:a@b.c", "$(rm -rf ~)",
	} {
		if err := validateWebURL(bad); err == nil {
			t.Errorf("%q should have been rejected", bad)
		}
	}
}

func TestOpenCommandsPreferWindowsOpenersUnderWSL(t *testing.T) {
	const u = "https://x.y"
	wsl := openCommands("linux", true, u)
	if want := [][]string{{"wslview", u}, {"explorer.exe", u}, {"xdg-open", u}}; !reflect.DeepEqual(wsl, want) {
		t.Errorf("wsl = %v, want %v", wsl, want)
	}
	if got := openCommands("linux", false, u); !reflect.DeepEqual(got, [][]string{{"xdg-open", u}}) {
		t.Errorf("plain linux = %v", got)
	}
	if got := openCommands("darwin", false, u); !reflect.DeepEqual(got, [][]string{{"open", u}}) {
		t.Errorf("darwin = %v", got)
	}
	if got := openCommands("windows", false, u); len(got) != 1 || got[0][0] != "rundll32" {
		t.Errorf("windows = %v", got)
	}
	if got := openCommands("plan9", false, u); got != nil {
		t.Errorf("unsupported platform = %v", got)
	}
}

func TestOpenURLRefusesNonWebURLsBeforeRunningAnything(t *testing.T) {
	if err := OpenURL("file:///etc/passwd"); err == nil {
		t.Error("a file:// URL must never reach the opener")
	}
}
