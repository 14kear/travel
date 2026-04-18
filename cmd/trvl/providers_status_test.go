package main

import (
	"testing"
	"time"

	"github.com/MikkoParkkola/trvl/internal/providers"
)

func TestClassifyProviderStatus(t *testing.T) {
	tests := []struct {
		name string
		cfg  providers.ProviderConfig
		want string
	}{
		{
			name: "healthy: recent success, no errors",
			cfg: providers.ProviderConfig{
				LastSuccess: time.Now().Add(-1 * time.Hour),
			},
			want: "healthy",
		},
		{
			name: "stale: success older than 24h, no errors",
			cfg: providers.ProviderConfig{
				LastSuccess: time.Now().Add(-48 * time.Hour),
			},
			want: "stale",
		},
		{
			name: "error: has error count and message",
			cfg: providers.ProviderConfig{
				LastSuccess: time.Now().Add(-1 * time.Hour),
				ErrorCount:  3,
				LastError:   "connection refused",
			},
			want: "error",
		},
		{
			name: "unconfigured: zero last_success, no errors",
			cfg:  providers.ProviderConfig{},
			want: "unconfigured",
		},
		{
			name: "error takes precedence over stale",
			cfg: providers.ProviderConfig{
				LastSuccess: time.Now().Add(-48 * time.Hour),
				ErrorCount:  1,
				LastError:   "timeout",
			},
			want: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyProviderStatus(&tt.cfg)
			if got != tt.want {
				t.Errorf("classifyProviderStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRelativeTimeStr(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{
			name: "zero time",
			t:    time.Time{},
			want: "-",
		},
		{
			name: "just now",
			t:    time.Now().Add(-10 * time.Second),
			want: "just now",
		},
		{
			name: "minutes ago",
			t:    time.Now().Add(-5 * time.Minute),
			want: "5m ago",
		},
		{
			name: "one minute ago",
			t:    time.Now().Add(-1 * time.Minute),
			want: "1m ago",
		},
		{
			name: "hours ago",
			t:    time.Now().Add(-3 * time.Hour),
			want: "3h ago",
		},
		{
			name: "one hour ago",
			t:    time.Now().Add(-1 * time.Hour),
			want: "1h ago",
		},
		{
			name: "days ago",
			t:    time.Now().Add(-72 * time.Hour),
			want: "3d ago",
		},
		{
			name: "one day ago",
			t:    time.Now().Add(-25 * time.Hour),
			want: "1d ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTimeStr(tt.t)
			if got != tt.want {
				t.Errorf("relativeTimeStr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			s:      "hello",
			maxLen: 80,
			want:   "hello",
		},
		{
			name:   "exact length unchanged",
			s:      "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string truncated with ellipsis",
			s:      "this is a very long error message that exceeds the maximum allowed length for display",
			maxLen: 30,
			want:   "this is a very long error m...",
		},
		{
			name:   "very short maxLen",
			s:      "hello",
			maxLen: 3,
			want:   "hel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateStr(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}
