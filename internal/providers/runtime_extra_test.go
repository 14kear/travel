package providers

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

// --- stripHTMLTags ---

func TestStripHTMLTags_NoTags(t *testing.T) {
	got := stripHTMLTags("Hello, World!")
	if got != "Hello, World!" {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestStripHTMLTags_SimpleTags(t *testing.T) {
	got := stripHTMLTags("<b>Hello</b> <i>World</i>")
	if got != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", got)
	}
}

func TestStripHTMLTags_BrTag(t *testing.T) {
	got := stripHTMLTags("Line1<br>Line2")
	if got != "Line1 Line2" {
		t.Errorf("expected 'Line1 Line2', got %q", got)
	}
}

func TestStripHTMLTags_BrSelfClosing(t *testing.T) {
	got := stripHTMLTags("Line1<br/>Line2<br />Line3")
	if got != "Line1 Line2 Line3" {
		t.Errorf("expected 'Line1 Line2 Line3', got %q", got)
	}
}

func TestStripHTMLTags_BrUpperCase(t *testing.T) {
	got := stripHTMLTags("A<BR>B")
	if got != "A B" {
		t.Errorf("expected 'A B', got %q", got)
	}
}

func TestStripHTMLTags_CollapseWhitespace(t *testing.T) {
	got := stripHTMLTags("<p>  Hello   World  </p>")
	if got != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", got)
	}
}

func TestStripHTMLTags_Empty(t *testing.T) {
	got := stripHTMLTags("")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestStripHTMLTags_NestedTags(t *testing.T) {
	got := stripHTMLTags("<div><span>Text</span></div>")
	if got != "Text" {
		t.Errorf("expected 'Text', got %q", got)
	}
}

// --- isAkamaiChallenge ---

func TestIsAkamaiChallenge_Not202(t *testing.T) {
	body := []byte("challenge.js window.aws reportChallengeError awswaf")
	if isAkamaiChallenge(http.StatusOK, body) {
		t.Error("expected false for 200 OK")
	}
	if isAkamaiChallenge(http.StatusForbidden, body) {
		t.Error("expected false for 403")
	}
}

func TestIsAkamaiChallenge_202WithJSON(t *testing.T) {
	body := []byte(`{"status":"ok"}`)
	if isAkamaiChallenge(http.StatusAccepted, body) {
		t.Error("expected false for 202 with JSON body")
	}
}

func TestIsAkamaiChallenge_202WithArrayJSON(t *testing.T) {
	body := []byte(`[{"id":1}]`)
	if isAkamaiChallenge(http.StatusAccepted, body) {
		t.Error("expected false for 202 with JSON array body")
	}
}

func TestIsAkamaiChallenge_202WithChallengeJS(t *testing.T) {
	body := []byte(`<html><script src="challenge.js"></script></html>`)
	if !isAkamaiChallenge(http.StatusAccepted, body) {
		t.Error("expected true for 202 with challenge.js")
	}
}

func TestIsAkamaiChallenge_202WithWindowAWS(t *testing.T) {
	body := []byte(`<html>window.aws = {}</html>`)
	if !isAkamaiChallenge(http.StatusAccepted, body) {
		t.Error("expected true for 202 with window.aws")
	}
}

func TestIsAkamaiChallenge_202WithReportChallengeError(t *testing.T) {
	body := []byte(`<script>reportChallengeError()</script>`)
	if !isAkamaiChallenge(http.StatusAccepted, body) {
		t.Error("expected true for 202 with reportChallengeError")
	}
}

func TestIsAkamaiChallenge_202WithAWSWAF(t *testing.T) {
	body := []byte(`<html>awswaf token present</html>`)
	if !isAkamaiChallenge(http.StatusAccepted, body) {
		t.Error("expected true for 202 with awswaf")
	}
}

func TestIsAkamaiChallenge_202WithNoMarkers(t *testing.T) {
	body := []byte(`<html><body>Normal async response</body></html>`)
	if isAkamaiChallenge(http.StatusAccepted, body) {
		t.Error("expected false for 202 without challenge markers")
	}
}

// --- needsBrowserCookieFallback ---

func TestNeedsBrowserCookieFallback_202(t *testing.T) {
	if !needsBrowserCookieFallback(http.StatusAccepted, 0, nil) {
		t.Error("expected true for 202")
	}
}

func TestNeedsBrowserCookieFallback_403(t *testing.T) {
	if !needsBrowserCookieFallback(http.StatusForbidden, 0, nil) {
		t.Error("expected true for 403")
	}
}

func TestNeedsBrowserCookieFallback_ZeroExtractions(t *testing.T) {
	extractions := map[string]Extraction{"token": {Pattern: `tok=(\w+)`}}
	if !needsBrowserCookieFallback(http.StatusOK, 0, extractions) {
		t.Error("expected true when extractions defined but 0 matched")
	}
}

func TestNeedsBrowserCookieFallback_SomeExtracted(t *testing.T) {
	extractions := map[string]Extraction{"token": {Pattern: `tok=(\w+)`}}
	if needsBrowserCookieFallback(http.StatusOK, 1, extractions) {
		t.Error("expected false when extractions succeeded")
	}
}

func TestNeedsBrowserCookieFallback_200NoExtractions(t *testing.T) {
	if needsBrowserCookieFallback(http.StatusOK, 0, nil) {
		t.Error("expected false for 200 with no extractions defined")
	}
}

// --- decompressBody ---

func TestDecompressBody_Identity(t *testing.T) {
	content := []byte("hello world")
	resp := &http.Response{
		Header: http.Header{},
		Body:   http.NoBody,
	}
	resp.Body = makeBody(content)
	resp.Header.Set("Content-Encoding", "identity")

	got, err := decompressBody(resp, 1024*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(got))
	}
}

func TestDecompressBody_Gzip(t *testing.T) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write([]byte("compressed content"))
	w.Close()

	resp := &http.Response{
		Header: http.Header{},
		Body:   makeBody(buf.Bytes()),
	}
	resp.Header.Set("Content-Encoding", "gzip")

	got, err := decompressBody(resp, 1024*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "compressed content" {
		t.Errorf("expected 'compressed content', got %q", string(got))
	}
}

func TestDecompressBody_GzipHeaderButNotGzip(t *testing.T) {
	// Content-Encoding says gzip but body is plain text — should return raw.
	content := []byte("not actually gzip")
	resp := &http.Response{
		Header: http.Header{},
		Body:   makeBody(content),
	}
	resp.Header.Set("Content-Encoding", "gzip")

	got, err := decompressBody(resp, 1024*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "not actually gzip" {
		t.Errorf("expected raw fallback, got %q", string(got))
	}
}

func TestDecompressBody_NoEncoding(t *testing.T) {
	content := []byte("plain body")
	resp := &http.Response{
		Header: http.Header{},
		Body:   makeBody(content),
	}
	// No Content-Encoding header.

	got, err := decompressBody(resp, 1024*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "plain body" {
		t.Errorf("expected 'plain body', got %q", string(got))
	}
}

func TestDecompressBody_Uncompressed(t *testing.T) {
	// When resp.Uncompressed=true, read raw even if Content-Encoding header set.
	content := []byte("already decompressed")
	resp := &http.Response{
		Header:       http.Header{},
		Body:         makeBody(content),
		Uncompressed: true,
	}
	resp.Header.Set("Content-Encoding", "gzip")

	got, err := decompressBody(resp, 1024*1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "already decompressed" {
		t.Errorf("expected 'already decompressed', got %q", string(got))
	}
}

// --- applyExtractions ---

func TestApplyExtractions_HeaderExtraction(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{"X-Token": []string{"tok_abc123"}},
		Body:   http.NoBody,
	}
	extractions := map[string]Extraction{
		"token": {Header: "X-Token", Pattern: `tok_(\w+)`},
	}
	authValues := map[string]string{}

	matched := applyExtractions(extractions, resp, nil, authValues)
	if matched != 1 {
		t.Errorf("expected 1 match, got %d", matched)
	}
	if authValues["token"] != "abc123" {
		t.Errorf("expected 'abc123', got %q", authValues["token"])
	}
}

func TestApplyExtractions_BodyExtraction(t *testing.T) {
	body := []byte(`<html>window.token = "xyz789";</html>`)
	resp := &http.Response{
		Header: http.Header{},
		Body:   http.NoBody,
	}
	extractions := map[string]Extraction{
		"tok": {Pattern: `window\.token = "([^"]+)"`},
	}
	authValues := map[string]string{}

	matched := applyExtractions(extractions, resp, body, authValues)
	if matched != 1 {
		t.Errorf("expected 1 match, got %d", matched)
	}
	if authValues["tok"] != "xyz789" {
		t.Errorf("expected 'xyz789', got %q", authValues["tok"])
	}
}

func TestApplyExtractions_DefaultFallback(t *testing.T) {
	body := []byte(`<html>no match here</html>`)
	resp := &http.Response{Header: http.Header{}, Body: http.NoBody}
	extractions := map[string]Extraction{
		"tok": {Pattern: `NEVER_FOUND=(\w+)`, Default: "fallback_val"},
	}
	authValues := map[string]string{}

	matched := applyExtractions(extractions, resp, body, authValues)
	if matched != 1 {
		t.Errorf("expected 1 match (via default), got %d", matched)
	}
	if authValues["tok"] != "fallback_val" {
		t.Errorf("expected 'fallback_val', got %q", authValues["tok"])
	}
}

func TestApplyExtractions_URLDeferred(t *testing.T) {
	// Extractions with URL set should be skipped by applyExtractions.
	body := []byte(`token=should_not_be_extracted`)
	resp := &http.Response{Header: http.Header{}, Body: http.NoBody}
	extractions := map[string]Extraction{
		"tok": {URL: "https://example.com/bundle.js", Pattern: `token=(\w+)`},
	}
	authValues := map[string]string{}

	matched := applyExtractions(extractions, resp, body, authValues)
	if matched != 0 {
		t.Errorf("expected 0 matches for URL-deferred extraction, got %d", matched)
	}
}

func TestApplyExtractions_InvalidRegex(t *testing.T) {
	resp := &http.Response{Header: http.Header{}, Body: http.NoBody}
	extractions := map[string]Extraction{
		"bad": {Pattern: `[invalid`},
	}
	authValues := map[string]string{}

	// Should not panic; returns 0.
	matched := applyExtractions(extractions, resp, []byte("data"), authValues)
	if matched != 0 {
		t.Errorf("expected 0 for invalid regex, got %d", matched)
	}
}

func TestApplyExtractions_CustomVariable(t *testing.T) {
	body := []byte(`csrf_token=abcdef`)
	resp := &http.Response{Header: http.Header{}, Body: http.NoBody}
	extractions := map[string]Extraction{
		"csrf": {Pattern: `csrf_token=(\w+)`, Variable: "csrf_var"},
	}
	authValues := map[string]string{}

	matched := applyExtractions(extractions, resp, body, authValues)
	if matched != 1 {
		t.Errorf("expected 1, got %d", matched)
	}
	if authValues["csrf_var"] != "abcdef" {
		t.Errorf("expected authValues['csrf_var']='abcdef', got %q", authValues["csrf_var"])
	}
}

// --- applyURLExtractions ---

func TestApplyURLExtractions_NilClient(t *testing.T) {
	extractions := map[string]Extraction{
		"tok": {URL: "http://example.com/bundle.js", Pattern: `tok=(\w+)`},
	}
	authValues := map[string]string{}
	matched := applyURLExtractions(context.Background(), nil, extractions, authValues)
	if matched != 0 {
		t.Errorf("expected 0 for nil client, got %d", matched)
	}
}

func TestApplyURLExtractions_SuccessfulFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `var api_key = "KEY_12345";`)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	extractions := map[string]Extraction{
		"api_key": {URL: srv.URL, Pattern: `api_key = "([^"]+)"`},
	}
	authValues := map[string]string{}

	matched := applyURLExtractions(context.Background(), client, extractions, authValues)
	if matched != 1 {
		t.Errorf("expected 1, got %d", matched)
	}
	if authValues["api_key"] != "KEY_12345" {
		t.Errorf("expected 'KEY_12345', got %q", authValues["api_key"])
	}
}

func TestApplyURLExtractions_VarSubstitution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `secret=MYSECRET`)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	extractions := map[string]Extraction{
		"secret": {URL: srv.URL + "?id=${bundle_id}", Pattern: `secret=(\w+)`},
	}
	// bundle_id resolved from prior extraction.
	authValues := map[string]string{"bundle_id": "42"}

	matched := applyURLExtractions(context.Background(), client, extractions, authValues)
	if matched != 1 {
		t.Errorf("expected 1, got %d", matched)
	}
	if authValues["secret"] != "MYSECRET" {
		t.Errorf("expected 'MYSECRET', got %q", authValues["secret"])
	}
}

func TestApplyURLExtractions_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `secret=WONT_MATCH`)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	extractions := map[string]Extraction{
		"secret": {URL: srv.URL, Pattern: `secret=(\w+)`},
	}
	authValues := map[string]string{}

	matched := applyURLExtractions(context.Background(), client, extractions, authValues)
	if matched != 0 {
		t.Errorf("expected 0 for non-2xx response, got %d", matched)
	}
}

func TestApplyURLExtractions_SkipsNonURLEntries(t *testing.T) {
	// Entries without URL field should be skipped.
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	extractions := map[string]Extraction{
		"body_only": {Pattern: `tok=(\w+)`}, // no URL
	}
	authValues := map[string]string{}

	matched := applyURLExtractions(context.Background(), client, extractions, authValues)
	if matched != 0 {
		t.Errorf("expected 0 for non-URL extraction, got %d", matched)
	}
}

func TestApplyURLExtractions_DefaultFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `no match here`)
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	extractions := map[string]Extraction{
		"tok": {URL: srv.URL, Pattern: `NEVER_FOUND=(\w+)`, Default: "default_tok"},
	}
	authValues := map[string]string{}

	matched := applyURLExtractions(context.Background(), client, extractions, authValues)
	if matched != 1 {
		t.Errorf("expected 1 (via default), got %d", matched)
	}
	if authValues["tok"] != "default_tok" {
		t.Errorf("expected 'default_tok', got %q", authValues["tok"])
	}
}

// makeBody creates an io.ReadCloser from bytes.
func makeBody(b []byte) interface{ Read([]byte) (int, error); Close() error } {
	return &nopCloser{bytes.NewReader(b)}
}

type nopCloser struct {
	*bytes.Reader
}

func (n *nopCloser) Close() error { return nil }
