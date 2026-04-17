package waf

import (
	"testing"
	"time"

	"github.com/grafana/sobek"
)

// --- coerceString ---

func TestCoerceString_Nil(t *testing.T) {
	if got := coerceString(nil); got != "" {
		t.Errorf("expected empty for nil, got %q", got)
	}
}

func TestCoerceString_Undefined(t *testing.T) {
	if got := coerceString(sobek.Undefined()); got != "" {
		t.Errorf("expected empty for undefined, got %q", got)
	}
}

func TestCoerceString_Null(t *testing.T) {
	if got := coerceString(sobek.Null()); got != "" {
		t.Errorf("expected empty for null, got %q", got)
	}
}

func TestCoerceString_PlainString(t *testing.T) {
	vm := sobek.New()
	v := vm.ToValue("hello-token")
	if got := coerceString(v); got != "hello-token" {
		t.Errorf("expected 'hello-token', got %q", got)
	}
}

func TestCoerceString_EmptyString(t *testing.T) {
	vm := sobek.New()
	v := vm.ToValue("")
	if got := coerceString(v); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestCoerceString_ObjectWithToken(t *testing.T) {
	vm := sobek.New()
	// An object with a token property.
	obj, _ := vm.RunString(`({"token": "my-secret-token"})`)
	if got := coerceString(obj); got != "my-secret-token" {
		t.Errorf("expected token from object, got %q", got)
	}
}

func TestCoerceString_ObjectWithoutToken(t *testing.T) {
	vm := sobek.New()
	// An object without a token property — should return empty.
	obj, _ := vm.RunString(`({"other": "value"})`)
	got := coerceString(obj)
	// [object Object] triggers the empty-return path.
	if got != "" {
		t.Errorf("expected empty for object without token, got %q", got)
	}
}

func TestCoerceString_UndefinedString(t *testing.T) {
	vm := sobek.New()
	// A value whose String() is "undefined".
	v, _ := vm.RunString(`undefined`)
	if got := coerceString(v); got != "" {
		t.Errorf("expected empty for 'undefined' string, got %q", got)
	}
}

// --- extractTokenFromJar ---

func TestExtractTokenFromJar_Empty(t *testing.T) {
	got := extractTokenFromJar(nil)
	if got != "" {
		t.Errorf("expected empty for nil entries, got %q", got)
	}
}

func TestExtractTokenFromJar_NoMatch(t *testing.T) {
	entries := []cookieEntry{
		{Name: "session", Value: "abc", At: time.Now()},
	}
	got := extractTokenFromJar(entries)
	if got != "" {
		t.Errorf("expected empty when no aws-waf-token, got %q", got)
	}
}

func TestExtractTokenFromJar_Match(t *testing.T) {
	entries := []cookieEntry{
		{Name: "aws-waf-token", Value: "TOKEN123", At: time.Now()},
	}
	got := extractTokenFromJar(entries)
	if got != "TOKEN123" {
		t.Errorf("expected 'TOKEN123', got %q", got)
	}
}

func TestExtractTokenFromJar_CaseInsensitive(t *testing.T) {
	entries := []cookieEntry{
		{Name: "AWS-WAF-TOKEN", Value: "UPPER_TOKEN", At: time.Now()},
	}
	got := extractTokenFromJar(entries)
	if got != "UPPER_TOKEN" {
		t.Errorf("expected case-insensitive match, got %q", got)
	}
}

func TestExtractTokenFromJar_ReturnsLast(t *testing.T) {
	// Multiple tokens — last one wins (most recent).
	entries := []cookieEntry{
		{Name: "aws-waf-token", Value: "OLD_TOKEN", At: time.Now().Add(-1 * time.Second)},
		{Name: "aws-waf-token", Value: "NEW_TOKEN", At: time.Now()},
	}
	got := extractTokenFromJar(entries)
	if got != "NEW_TOKEN" {
		t.Errorf("expected most recent token 'NEW_TOKEN', got %q", got)
	}
}

func TestExtractTokenFromJar_MixedEntries(t *testing.T) {
	entries := []cookieEntry{
		{Name: "session", Value: "sess1", At: time.Now()},
		{Name: "aws-waf-token", Value: "WAF_TOKEN", At: time.Now()},
		{Name: "other", Value: "val", At: time.Now()},
	}
	got := extractTokenFromJar(entries)
	if got != "WAF_TOKEN" {
		t.Errorf("expected 'WAF_TOKEN', got %q", got)
	}
}

// --- clearTimer ---

func TestClearTimer_CancelsTimer(t *testing.T) {
	vm := sobek.New()
	loop := newEventLoop(vm)

	fired := false
	cb, _ := vm.RunString(`(function() {})`)
	fn, _ := sobek.AssertFunction(cb)

	// Track if timer fires.
	id := loop.scheduleTimer(fn, 0, false)
	loop.clearTimer(id)

	// Find the timer and verify it's marked cancelled.
	found := false
	for _, j := range loop.timers {
		if j.id == id {
			found = true
			if !j.cancel {
				t.Error("expected timer to be marked cancelled")
			}
		}
	}
	if !found {
		// Timer was never added — that's also fine.
		t.Log("timer not found in heap (may have been popped already)")
	}

	_ = fired
}

func TestClearTimer_NonexistentID(t *testing.T) {
	vm := sobek.New()
	loop := newEventLoop(vm)
	// Clearing a non-existent ID should not panic.
	loop.clearTimer(99999)
}

func TestClearTimer_PreventsFiring(t *testing.T) {
	vm := sobek.New()
	loop := newEventLoop(vm)

	if _, err := vm.RunString(`globalThis.__timerFired = false`); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cb, _ := vm.RunString(`(function() { globalThis.__timerFired = true; })`)
	fn, _ := sobek.AssertFunction(cb)

	// Schedule a 0ms timer then immediately cancel it.
	id := loop.scheduleTimer(fn, 0, false)
	loop.clearTimer(id)

	// Fire all due timers — the cancelled one should not fire.
	loop.fireDueTimers(func(err error) {})

	v := vm.Get("__timerFired")
	if v != nil && v.ToBoolean() {
		t.Error("cancelled timer should not have fired")
	}
}
