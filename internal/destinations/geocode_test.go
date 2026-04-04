package destinations

import "testing"

func TestExtractLocality(t *testing.T) {
	tests := []struct {
		name    string
		address map[string]string
		want    string
	}{
		{
			name:    "prefers city",
			address: map[string]string{"city": "Paris", "county": "Ile-de-France"},
			want:    "Paris",
		},
		{
			name:    "falls back to town",
			address: map[string]string{"town": "Reading"},
			want:    "Reading",
		},
		{
			name:    "falls back to county",
			address: map[string]string{"county": "Greater London"},
			want:    "Greater London",
		},
		{
			name:    "missing locality",
			address: map[string]string{"country": "Finland"},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractLocality(tt.address); got != tt.want {
				t.Fatalf("extractLocality(%v) = %q, want %q", tt.address, got, tt.want)
			}
		})
	}
}
