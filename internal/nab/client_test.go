package nab

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewReturnsErrNotAvailableWhenNabMissing(t *testing.T) {
	origLookPath := lookPath
	lookPath = func(string) (string, error) {
		return "", exec.ErrNotFound
	}
	t.Cleanup(func() {
		lookPath = origLookPath
	})

	client, err := New()
	if client != nil {
		t.Fatalf("client = %#v, want nil", client)
	}
	if !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("err = %v, want ErrNotAvailable", err)
	}
}

func TestClientFetchHTMLDefaultsToAutoCookies(t *testing.T) {
	script := writeMockNab(t, `#!/bin/sh
[ "$1" = "fetch" ] || { echo "bad subcommand" >&2; exit 11; }
[ "$2" = "https://example.com" ] || { echo "bad url" >&2; exit 12; }
[ "$3" = "--format" ] || { echo "missing --format" >&2; exit 13; }
[ "$4" = "json" ] || { echo "missing json format" >&2; exit 14; }
[ "$5" = "--raw-html" ] || { echo "missing --raw-html" >&2; exit 15; }
[ "$6" = "--cookies" ] || { echo "missing --cookies" >&2; exit 16; }
[ "$7" = "auto" ] || { echo "expected auto cookies" >&2; exit 17; }
[ "$8" = "--no-save" ] || { echo "missing --no-save" >&2; exit 18; }
printf '%s' '{"status":200,"markdown":"<html>ok</html>"}'
`)

	client := &Client{path: script}
	html, err := client.FetchHTML(context.Background(), "https://example.com", "")
	if err != nil {
		t.Fatalf("FetchHTML returned error: %v", err)
	}
	if got := string(html); got != "<html>ok</html>" {
		t.Fatalf("html = %q, want %q", got, "<html>ok</html>")
	}
}

func TestClientFetchHTMLRejectsNon200(t *testing.T) {
	script := writeMockNab(t, `#!/bin/sh
printf '%s' '{"status":403,"markdown":"blocked"}'
`)

	client := &Client{path: script}
	_, err := client.FetchHTML(context.Background(), "https://example.com", "auto")
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("err = %v, want status 403 error", err)
	}
}

func writeMockNab(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "nab")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write mock nab: %v", err)
	}
	return path
}
