package destinations

import (
	"testing"
)

func TestWeatherCodeDescription_AllBranches(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "Clear sky"},
		{1, "Mainly clear"},
		{2, "Partly cloudy"},
		{3, "Overcast"},
		{45, "Fog"},
		{48, "Fog"},
		{51, "Drizzle"},
		{55, "Drizzle"},
		{56, "Freezing drizzle"},
		{57, "Freezing drizzle"},
		{61, "Rain"},
		{65, "Rain"},
		{66, "Freezing rain"},
		{67, "Freezing rain"},
		{71, "Snow"},
		{77, "Snow"},
		{80, "Rain showers"},
		{82, "Rain showers"},
		{85, "Snow showers"},
		{86, "Snow showers"},
		{95, "Thunderstorm"},
		{99, "Thunderstorm"},
		{-1, "Unknown"},
		{100, "Unknown"},
		{10, "Unknown"},
	}
	for _, tt := range tests {
		got := weatherCodeDescription(tt.code)
		if got != tt.want {
			t.Errorf("weatherCodeDescription(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
