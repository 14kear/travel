package nab

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// skipOnWindowsCov2 short-circuits tests that rely on a POSIX-shell mock nab binary.
func skipOnWindowsCov2(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-shell mock nab binary is not portable to Windows")
	}
}

// --- LookupPath (75% -> cover success path) ---

func TestLookupPath_Success(t *testing.T) {
	origLookPath := lookPath
	lookPath = func(name string) (string, error) {
		return "/usr/local/bin/nab", nil
	}
	t.Cleanup(func() { lookPath = origLookPath })

	path, err := LookupPath()
	if err != nil {
		t.Fatalf("LookupPath: %v", err)
	}
	if path != "/usr/local/bin/nab" {
		t.Errorf("path = %q, want /usr/local/bin/nab", path)
	}
}

func TestLookupPath_NotFound(t *testing.T) {
	origLookPath := lookPath
	lookPath = func(name string) (string, error) {
		return "", exec.ErrNotFound
	}
	t.Cleanup(func() { lookPath = origLookPath })

	_, err := LookupPath()
	if !errors.Is(err, ErrNotAvailable) {
		t.Errorf("expected ErrNotAvailable, got %v", err)
	}
}

// --- New (75% -> cover success path) ---

func TestNew_Success(t *testing.T) {
	origLookPath := lookPath
	lookPath = func(name string) (string, error) {
		return "/mock/nab", nil
	}
	t.Cleanup(func() { lookPath = origLookPath })

	client, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if !client.Available() {
		t.Error("client should be available")
	}
}

// --- Available edge cases ---

func TestAvailable_NilClient(t *testing.T) {
	var c *Client
	if c.Available() {
		t.Error("nil client should not be available")
	}
}

func TestAvailable_EmptyPath(t *testing.T) {
	c := &Client{path: ""}
	if c.Available() {
		t.Error("client with empty path should not be available")
	}
}

// --- Fetch (69.2% -> cover error paths) ---

func TestFetch_NotAvailable(t *testing.T) {
	c := &Client{path: ""}
	_, err := c.Fetch(context.Background(), "https://example.com", FetchOptions{})
	if !errors.Is(err, ErrNotAvailable) {
		t.Errorf("expected ErrNotAvailable, got %v", err)
	}
}

func TestFetch_BadJSON(t *testing.T) {
	skipOnWindowsCov2(t)
	script := writeMockNabCov2(t, `#!/bin/sh
printf '%s' 'not json at all'
`)
	client := &Client{path: script}
	_, err := client.Fetch(context.Background(), "https://example.com", FetchOptions{})
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
	if !strings.Contains(err.Error(), "json") {
		t.Errorf("expected json error, got %v", err)
	}
}

func TestFetch_ExitError(t *testing.T) {
	skipOnWindowsCov2(t)
	script := writeMockNabCov2(t, `#!/bin/sh
echo "something went wrong" >&2
exit 1
`)
	client := &Client{path: script}
	_, err := client.Fetch(context.Background(), "https://example.com", FetchOptions{})
	if err == nil {
		t.Fatal("expected error from exit code 1")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("expected stderr message in error, got %v", err)
	}
}

func TestFetch_ExitErrorNoStderr(t *testing.T) {
	skipOnWindowsCov2(t)
	script := writeMockNabCov2(t, `#!/bin/sh
exit 1
`)
	client := &Client{path: script}
	_, err := client.Fetch(context.Background(), "https://example.com", FetchOptions{})
	if err == nil {
		t.Fatal("expected error from exit code 1")
	}
	// Without stderr, should get a generic error.
	if !strings.Contains(err.Error(), "nab fetch") {
		t.Errorf("expected 'nab fetch' in error, got %v", err)
	}
}

func TestFetch_GETMethod(t *testing.T) {
	skipOnWindowsCov2(t)
	// GET method should NOT add -X flag.
	script := writeMockNabCov2(t, `#!/bin/sh
# Verify no -X flag present for GET.
for arg in "$@"; do
  if [ "$arg" = "-X" ]; then
    echo "unexpected -X flag" >&2
    exit 1
  fi
done
printf '%s' '{"status":200,"markdown":"<html>get</html>"}'
`)
	client := &Client{path: script}
	body, err := client.Fetch(context.Background(), "https://example.com", FetchOptions{
		Method: "GET",
	})
	if err != nil {
		t.Fatalf("Fetch GET: %v", err)
	}
	if string(body) != "<html>get</html>" {
		t.Errorf("body = %q", body)
	}
}

func TestFetch_CaseInsensitiveGET(t *testing.T) {
	skipOnWindowsCov2(t)
	// "get" (lowercase) should also not add -X flag.
	script := writeMockNabCov2(t, `#!/bin/sh
for arg in "$@"; do
  if [ "$arg" = "-X" ]; then
    echo "unexpected -X flag for lowercase get" >&2
    exit 1
  fi
done
printf '%s' '{"status":200,"markdown":"ok"}'
`)
	client := &Client{path: script}
	body, err := client.Fetch(context.Background(), "https://example.com", FetchOptions{
		Method: "get",
	})
	if err != nil {
		t.Fatalf("Fetch get: %v", err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q", body)
	}
}

func TestFetch_SpecificBrowser(t *testing.T) {
	skipOnWindowsCov2(t)
	script := writeMockNabCov2(t, `#!/bin/sh
# Check that --cookies uses the specified browser.
found_cookies=0
next_is_browser=0
for arg in "$@"; do
  if [ "$next_is_browser" = "1" ]; then
    if [ "$arg" != "chrome" ]; then
      echo "expected chrome browser, got $arg" >&2
      exit 1
    fi
    found_cookies=1
    next_is_browser=0
  fi
  if [ "$arg" = "--cookies" ]; then
    next_is_browser=1
  fi
done
if [ "$found_cookies" = "0" ]; then
  echo "missing --cookies flag" >&2
  exit 1
fi
printf '%s' '{"status":200,"markdown":"chrome-ok"}'
`)
	client := &Client{path: script}
	body, err := client.Fetch(context.Background(), "https://example.com", FetchOptions{
		Browser: "chrome",
	})
	if err != nil {
		t.Fatalf("Fetch with chrome browser: %v", err)
	}
	if string(body) != "chrome-ok" {
		t.Errorf("body = %q", body)
	}
}

func TestFetch_ContextCanceled(t *testing.T) {
	skipOnWindowsCov2(t)
	script := writeMockNabCov2(t, `#!/bin/sh
sleep 10
`)
	client := &Client{path: script}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Fetch(ctx, "https://example.com", FetchOptions{})
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func writeMockNabCov2(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nab")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write mock nab: %v", err)
	}
	return path
}
