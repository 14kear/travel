package waf

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/sobek"
)

// --- resolveURL (27.3% -> cover all branches) ---

func TestResolveURL_EmptyTarget(t *testing.T) {
	got := resolveURL("https://example.com", "")
	if got != "https://example.com" {
		t.Errorf("empty target: got %q, want origin", got)
	}
}

func TestResolveURL_AbsoluteHTTPS(t *testing.T) {
	got := resolveURL("https://example.com", "https://other.com/path")
	if got != "https://other.com/path" {
		t.Errorf("absolute https: got %q", got)
	}
}

func TestResolveURL_AbsoluteHTTP(t *testing.T) {
	got := resolveURL("https://example.com", "http://other.com/path")
	if got != "http://other.com/path" {
		t.Errorf("absolute http: got %q", got)
	}
}

func TestResolveURL_RelativePath(t *testing.T) {
	got := resolveURL("https://example.com", "/challenge/token")
	if got != "https://example.com/challenge/token" {
		t.Errorf("relative path: got %q", got)
	}
}

func TestResolveURL_RelativeNoSlash(t *testing.T) {
	got := resolveURL("https://example.com/base/", "script.js")
	if got != "https://example.com/base/script.js" {
		t.Errorf("relative no slash: got %q", got)
	}
}

func TestResolveURL_InvalidOrigin(t *testing.T) {
	got := resolveURL("://bad", "/path")
	// Should return target unchanged on parse error.
	if got != "/path" {
		t.Errorf("invalid origin: got %q, want /path", got)
	}
}

func TestResolveURL_InvalidTarget(t *testing.T) {
	got := resolveURL("https://example.com", "://bad-ref")
	// Should return target unchanged on parse error.
	if got != "://bad-ref" {
		t.Errorf("invalid target: got %q", got)
	}
}

// --- readByteView (38.1% -> cover nil/undefined/null/string/array paths) ---

func TestReadByteView_Nil(t *testing.T) {
	vm := sobek.New()
	got := readByteView(vm, nil)
	if got != nil {
		t.Errorf("nil input: got %v, want nil", got)
	}
}

func TestReadByteView_Undefined(t *testing.T) {
	vm := sobek.New()
	got := readByteView(vm, sobek.Undefined())
	if got != nil {
		t.Errorf("undefined: got %v, want nil", got)
	}
}

func TestReadByteView_Null(t *testing.T) {
	vm := sobek.New()
	got := readByteView(vm, sobek.Null())
	if got != nil {
		t.Errorf("null: got %v, want nil", got)
	}
}

func TestReadByteView_String(t *testing.T) {
	vm := sobek.New()
	got := readByteView(vm, vm.ToValue("hello"))
	if string(got) != "hello" {
		t.Errorf("string: got %q, want hello", got)
	}
}

func TestReadByteView_NumberArray(t *testing.T) {
	vm := sobek.New()
	v, err := vm.RunString(`[72, 101, 108, 108, 111]`) // "Hello"
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	got := readByteView(vm, v)
	if string(got) != "Hello" {
		t.Errorf("number array: got %q, want Hello", got)
	}
}

func TestReadByteView_Uint8Array(t *testing.T) {
	vm := sobek.New()
	v, err := vm.RunString(`new Uint8Array([65, 66, 67])`) // "ABC"
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	got := readByteView(vm, v)
	if string(got) != "ABC" {
		t.Errorf("Uint8Array: got %q, want ABC", got)
	}
}

func TestReadByteView_ObjectNoLength(t *testing.T) {
	vm := sobek.New()
	v, err := vm.RunString(`({})`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	got := readByteView(vm, v)
	if got != nil {
		t.Errorf("object without length: got %v, want nil", got)
	}
}

// --- cryptoRandom (0%) ---

func TestCryptoRandom_Uint8Array(t *testing.T) {
	vm, _, _ := newTestHost(t, http.DefaultClient)

	v, err := vm.RunString(`
		var arr = new Uint8Array(16);
		crypto.getRandomValues(arr);
		var nonZero = 0;
		for (var i = 0; i < arr.length; i++) {
			if (arr[i] !== 0) nonZero++;
		}
		nonZero > 0;
	`)
	if err != nil {
		t.Fatalf("cryptoRandom: %v", err)
	}
	if !v.ToBoolean() {
		t.Error("expected at least one non-zero byte from crypto.getRandomValues")
	}
}

func TestCryptoRandom_EmptyArray(t *testing.T) {
	vm, _, _ := newTestHost(t, http.DefaultClient)

	_, err := vm.RunString(`
		var arr = new Uint8Array(0);
		crypto.getRandomValues(arr);
	`)
	if err != nil {
		t.Fatalf("cryptoRandom empty: %v", err)
	}
}

// --- jsClearTimeout (0%) ---

func TestJsClearTimeout_ViaStubs(t *testing.T) {
	vm, loop, _ := newTestHost(t, http.DefaultClient)

	// Set a timer via JS, then clear it, verify it doesn't fire.
	if _, err := vm.RunString(`globalThis.__cleared = false`); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := vm.RunString(`
		var id = setTimeout(function() { globalThis.__cleared = true; }, 0);
		clearTimeout(id);
	`)
	if err != nil {
		t.Fatalf("set/clear timeout: %v", err)
	}

	// Run the loop briefly to let any queued timers fire.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = loop.run(ctx, nil)

	v := vm.Get("__cleared")
	if v != nil && v.ToBoolean() {
		t.Error("cleared timeout should not have fired")
	}
}

func TestJsClearTimeout_NoArgs(t *testing.T) {
	vm, _, _ := newTestHost(t, http.DefaultClient)

	// clearTimeout with no args should not panic.
	_, err := vm.RunString(`clearTimeout()`)
	if err != nil {
		t.Fatalf("clearTimeout() with no args threw: %v", err)
	}
}

// --- trackPending edge cases ---

func TestTrackPending_NegativeClampToZero(t *testing.T) {
	vm := sobek.New()
	loop := newEventLoop(vm)

	loop.trackPending(-5)
	if loop.pending != 0 {
		t.Errorf("pending = %d, want 0 (clamped from negative)", loop.pending)
	}
}

func TestTrackPending_IncrementDecrement(t *testing.T) {
	vm := sobek.New()
	loop := newEventLoop(vm)

	loop.trackPending(+1)
	loop.trackPending(+1)
	if loop.pending != 2 {
		t.Errorf("pending = %d, want 2", loop.pending)
	}
	loop.trackPending(-1)
	if loop.pending != 1 {
		t.Errorf("pending = %d, want 1", loop.pending)
	}
}

// --- enqueueFromGo when stopped ---

func TestEnqueueFromGo_WhenStopped(t *testing.T) {
	vm := sobek.New()
	loop := newEventLoop(vm)

	// Mark the loop as stopped.
	loop.mu.Lock()
	loop.stopped = true
	loop.mu.Unlock()

	// Should not panic or block.
	loop.enqueueFromGo(func() error {
		t.Error("callback should not be called when loop is stopped")
		return nil
	})
}

// --- digest edge cases (57.1% -> cover SHA-384, SHA-512, unsupported) ---

func TestDigest_SHA384(t *testing.T) {
	vm, loop, _ := newTestHost(t, http.DefaultClient)

	_, err := vm.RunString(`
		globalThis.__hash384 = null;
		var data = new TextEncoder().encode("test");
		crypto.subtle.digest("SHA-384", data).then(function(buf) {
			var u = new Uint8Array(buf);
			globalThis.__hash384 = u.length;
		});
	`)
	if err != nil {
		t.Fatalf("digest SHA-384 setup: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := func() bool {
		v := vm.Get("__hash384")
		return v != nil && !sobek.IsUndefined(v) && !sobek.IsNull(v)
	}
	if err := loop.run(ctx, done); err != nil {
		t.Fatalf("loop.run: %v", err)
	}

	v := vm.Get("__hash384")
	if v == nil || v.ToInteger() != 48 {
		t.Errorf("SHA-384 digest length: got %v, want 48", v)
	}
}

func TestDigest_SHA512(t *testing.T) {
	vm, loop, _ := newTestHost(t, http.DefaultClient)

	_, err := vm.RunString(`
		globalThis.__hash512 = null;
		var data = new TextEncoder().encode("test");
		crypto.subtle.digest("SHA-512", data).then(function(buf) {
			var u = new Uint8Array(buf);
			globalThis.__hash512 = u.length;
		});
	`)
	if err != nil {
		t.Fatalf("digest SHA-512 setup: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := func() bool {
		v := vm.Get("__hash512")
		return v != nil && !sobek.IsUndefined(v) && !sobek.IsNull(v)
	}
	if err := loop.run(ctx, done); err != nil {
		t.Fatalf("loop.run: %v", err)
	}

	v := vm.Get("__hash512")
	if v == nil || v.ToInteger() != 64 {
		t.Errorf("SHA-512 digest length: got %v, want 64", v)
	}
}

// --- jsSetTimeout edge cases ---

func TestJsSetTimeout_NoArgs(t *testing.T) {
	vm, _, _ := newTestHost(t, http.DefaultClient)

	v, err := vm.RunString(`setTimeout()`)
	if err != nil {
		t.Fatalf("setTimeout() no args: %v", err)
	}
	// Should return 0 for no-arg call.
	if v.ToInteger() != 0 {
		t.Errorf("setTimeout() = %v, want 0", v)
	}
}

func TestJsSetTimeout_NonFunction(t *testing.T) {
	vm, _, _ := newTestHost(t, http.DefaultClient)

	// Passing non-function should return 0 (not throw).
	v, err := vm.RunString(`setTimeout("not a function", 100)`)
	if err != nil {
		// Some JS engines do throw for non-callable; that's acceptable too.
		return
	}
	if v.ToInteger() != 0 {
		t.Errorf("setTimeout(string) = %v, want 0", v)
	}
}

func TestJsSetTimeout_NegativeDelay(t *testing.T) {
	vm, loop, _ := newTestHost(t, http.DefaultClient)

	_, err := vm.RunString(`
		globalThis.__negDelay = false;
		setTimeout(function() { globalThis.__negDelay = true; }, -100);
	`)
	if err != nil {
		t.Fatalf("setTimeout negative delay: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = loop.run(ctx, nil)

	v := vm.Get("__negDelay")
	if v == nil || !v.ToBoolean() {
		t.Error("negative delay should fire immediately (clamped to 0)")
	}
}

// --- jsSetCookie edge cases ---

func TestJsSetCookie_InsufficientArgs(t *testing.T) {
	vm, _, _ := newTestHost(t, http.DefaultClient)

	// Calling __goSetCookie with fewer than 3 args should return undefined, not panic.
	_, err := vm.RunString(`__goSetCookie("name", "val")`)
	if err != nil {
		t.Fatalf("jsSetCookie with 2 args: %v", err)
	}
}

// --- newUint8Array fallback ---

func TestNewUint8Array_Normal(t *testing.T) {
	vm := sobek.New()
	src := []byte{1, 2, 3, 4, 5}
	v := newUint8Array(vm, src)
	if v == nil || sobek.IsUndefined(v) {
		t.Fatal("expected non-nil Uint8Array")
	}

	// Verify length.
	obj := v.ToObject(vm)
	length := obj.Get("length")
	if length == nil || length.ToInteger() != 5 {
		t.Errorf("Uint8Array length: got %v, want 5", length)
	}
}

// --- buildCookie edge cases ---

func TestBuildCookie_NoHost(t *testing.T) {
	c := buildCookie("not-a-url", "TOKEN")
	if c.Name != "aws-waf-token" {
		t.Errorf("Name = %q", c.Name)
	}
	if c.Value != "TOKEN" {
		t.Errorf("Value = %q", c.Value)
	}
	// Domain should be empty for unparseable URL.
}

func TestBuildCookie_WithWWW(t *testing.T) {
	c := buildCookie("https://www.example.com/page", "T")
	if c.Domain != ".example.com" {
		t.Errorf("Domain = %q, want .example.com", c.Domain)
	}
}

func TestBuildCookie_WithoutWWW(t *testing.T) {
	c := buildCookie("https://api.example.com/page", "T")
	if c.Domain != ".api.example.com" {
		t.Errorf("Domain = %q, want .api.example.com", c.Domain)
	}
}

// --- parseChallengePage edge cases ---

func TestParseChallengePage_EmptyURL(t *testing.T) {
	body := `<script src="https://cdn.awswaf.com/challenge.js"></script>`
	info, err := parseChallengePage("", body)
	if err != nil {
		t.Fatalf("parseChallengePage empty URL: %v", err)
	}
	if info.origin != "" {
		t.Errorf("origin = %q, want empty for empty URL", info.origin)
	}
}

func TestParseChallengePage_NoGokuProps(t *testing.T) {
	body := `<script src="https://cdn.awswaf.com/challenge.js"></script>`
	info, err := parseChallengePage("https://example.com/", body)
	if err != nil {
		t.Fatalf("parseChallengePage: %v", err)
	}
	if info.gokuProps != "" {
		t.Errorf("gokuProps = %q, want empty", info.gokuProps)
	}
}

func TestParseChallengePage_NoScript(t *testing.T) {
	body := `<html><body>No challenge here</body></html>`
	_, err := parseChallengePage("https://example.com/", body)
	if err == nil {
		t.Error("expected ErrNoChallenge")
	}
}

// --- SolveAWSWAF options defaults ---

func TestSolveAWSWAF_DefaultOptions(t *testing.T) {
	// opts=nil should use defaults without panic.
	body := `<html><body>No challenge</body></html>`
	_, err := SolveAWSWAF(context.Background(), http.DefaultClient, "https://example.com/", body, nil)
	// Should fail with ErrNoChallenge, not panic.
	if err == nil {
		t.Error("expected error")
	}
}
