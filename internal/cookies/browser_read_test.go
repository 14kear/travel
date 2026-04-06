package cookies

import (
	"context"
	"testing"
	"time"
)

func TestSkipBrowserRead(t *testing.T) {
	orig := SkipBrowserRead
	SkipBrowserRead = true
	defer func() { SkipBrowserRead = orig }()

	_, err := BrowserReadPage(context.Background(), "https://example.com", 1)
	if err == nil {
		t.Fatal("expected error when SkipBrowserRead=true, got nil")
	}
}

func TestBrowserReadPageCachedHit(t *testing.T) {
	// Populate the cache manually then verify we get the cached value back
	// without BrowserReadPage being called (which would fail in CI).
	const testURL = "https://example.com/cached-test"
	const testText = "cached page content"

	browserPageCache.Lock()
	browserPageCache.entries[testURL] = browserCacheEntry{
		text:    testText,
		expires: time.Now().Add(5 * time.Minute),
	}
	browserPageCache.Unlock()
	defer func() {
		browserPageCache.Lock()
		delete(browserPageCache.entries, testURL)
		browserPageCache.Unlock()
	}()

	// Also ensure SkipBrowserRead=true so any accidental fallthrough to
	// the real BrowserReadPage fails loudly rather than opening a browser.
	orig := SkipBrowserRead
	SkipBrowserRead = true
	defer func() { SkipBrowserRead = orig }()

	got, err := BrowserReadPageCached(context.Background(), testURL, 1, 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error from cache hit: %v", err)
	}
	if got != testText {
		t.Errorf("got %q, want %q", got, testText)
	}
}

func TestBrowserReadPageCachedExpiry(t *testing.T) {
	// Populate the cache with an already-expired entry.
	const testURL = "https://example.com/expired-test"
	const staleText = "stale content"

	browserPageCache.Lock()
	browserPageCache.entries[testURL] = browserCacheEntry{
		text:    staleText,
		expires: time.Now().Add(-1 * time.Second), // already expired
	}
	browserPageCache.Unlock()
	defer func() {
		browserPageCache.Lock()
		delete(browserPageCache.entries, testURL)
		browserPageCache.Unlock()
	}()

	// SkipBrowserRead=true ensures BrowserReadPage returns an error (rather than
	// opening a browser) so we can confirm the cache was bypassed.
	orig := SkipBrowserRead
	SkipBrowserRead = true
	defer func() { SkipBrowserRead = orig }()

	_, err := BrowserReadPageCached(context.Background(), testURL, 1, 5*time.Minute)
	if err == nil {
		t.Fatal("expected error because cache entry expired and browser read is disabled, got nil")
	}
}
