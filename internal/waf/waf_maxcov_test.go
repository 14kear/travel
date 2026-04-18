package waf

import (
	"testing"
	"time"
)

func TestTimerHeapLess(t *testing.T) {
	now := time.Now()
	h := timerHeap{
		{id: 1, when: now},
		{id: 2, when: now.Add(time.Second)},
	}

	if !h.Less(0, 1) {
		t.Error("earlier job should be less than later job")
	}
	if h.Less(1, 0) {
		t.Error("later job should not be less than earlier job")
	}
}
