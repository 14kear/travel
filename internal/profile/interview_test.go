package profile

import (
	"testing"

	"github.com/MikkoParkkola/trvl/internal/preferences"
)

func TestInterviewQuestionsNoProfile(t *testing.T) {
	questions := InterviewQuestions(nil, nil)

	// With no profile and no prefs, all questions should be asked.
	if len(questions) < 5 {
		t.Errorf("expected >= 5 questions with no context, got %d", len(questions))
	}

	// Verify key questions are present.
	keys := make(map[string]bool)
	for _, q := range questions {
		keys[q.Key] = true
	}

	required := []string{"travellers", "dates_fixed", "budget", "luggage", "purpose", "priority"}
	for _, key := range required {
		if !keys[key] {
			t.Errorf("missing question key: %s", key)
		}
	}
}

func TestInterviewQuestionsWithBudgetPrefs(t *testing.T) {
	prefs := &preferences.Preferences{
		BudgetPerNightMax: 200,
		BudgetFlightMax:   500,
	}

	questions := InterviewQuestions(nil, prefs)

	// Budget question should be skipped.
	for _, q := range questions {
		if q.Key == "budget" {
			t.Error("budget question should be skipped when prefs have budget")
		}
	}
}

func TestInterviewQuestionsWithLuggagePrefs(t *testing.T) {
	prefs := &preferences.Preferences{
		CarryOnOnly:  true,
		HomeAirports: []string{"HEL"}, // signals prefs are configured
	}

	questions := InterviewQuestions(nil, prefs)

	for _, q := range questions {
		if q.Key == "luggage" {
			t.Error("luggage question should be skipped when prefs have carry_on_only")
		}
	}
}

func TestInterviewQuestionsWithAccomPrefs(t *testing.T) {
	prefs := &preferences.Preferences{
		NoDormitories: true,
		EnSuiteOnly:   true,
	}

	questions := InterviewQuestions(nil, prefs)

	for _, q := range questions {
		if q.Key == "accom_needs" {
			t.Error("accom_needs question should be skipped when prefs have accommodation settings")
		}
	}
}

func TestInterviewQuestionsWithDealTolerance(t *testing.T) {
	prefs := &preferences.Preferences{
		DealTolerance: "comfort",
	}

	questions := InterviewQuestions(nil, prefs)

	for _, q := range questions {
		if q.Key == "priority" {
			t.Error("priority question should be skipped when prefs have deal_tolerance")
		}
	}
}

func TestInterviewQuestionsAlwaysAsksDates(t *testing.T) {
	// Even with full profile + prefs, dates should always be asked.
	prefs := &preferences.Preferences{
		BudgetPerNightMax: 200,
		CarryOnOnly:       true,
		HomeAirports:      []string{"HEL"},
		NoDormitories:     true,
		DealTolerance:     "balanced",
	}
	prof := &TravelProfile{
		TotalTrips: 20,
		Bookings:   make([]Booking, 20),
	}

	questions := InterviewQuestions(prof, prefs)

	foundDates := false
	for _, q := range questions {
		if q.Key == "dates_fixed" {
			foundDates = true
		}
	}
	if !foundDates {
		t.Error("dates_fixed question should always be present")
	}
}

func TestInterviewQuestionsPurposeDefault(t *testing.T) {
	prefs := &preferences.Preferences{
		FastWifiNeeded: true,
	}

	questions := InterviewQuestions(nil, prefs)

	for _, q := range questions {
		if q.Key == "purpose" {
			if q.Default != "remote_work" {
				t.Errorf("purpose default = %q, want remote_work (fast_wifi_needed is set)", q.Default)
			}
		}
	}
}

func TestInterviewQuestionsPriorityDefault(t *testing.T) {
	prof := &TravelProfile{
		BudgetTier: "budget",
	}

	questions := InterviewQuestions(prof, nil)

	for _, q := range questions {
		if q.Key == "priority" {
			if q.Default != "price" {
				t.Errorf("priority default = %q, want price (budget tier)", q.Default)
			}
		}
	}
}

func TestInterviewQuestionTypes(t *testing.T) {
	questions := InterviewQuestions(nil, nil)

	for _, q := range questions {
		switch q.Type {
		case "number", "choice", "multi_choice", "text":
			// OK
		default:
			t.Errorf("question %q has unexpected type %q", q.Key, q.Type)
		}

		if q.Type == "choice" || q.Type == "multi_choice" {
			if len(q.Options) == 0 {
				t.Errorf("question %q of type %s has no options", q.Key, q.Type)
			}
		}
	}
}

func TestIntToStr(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{100, "100"},
		{-5, "-5"},
	}
	for _, tt := range tests {
		got := intToStr(tt.n)
		if got != tt.want {
			t.Errorf("intToStr(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestFormatCurrency(t *testing.T) {
	if got := formatCurrency(199.50); got != "199" {
		t.Errorf("formatCurrency(199.50) = %q, want 199", got)
	}
}

func TestIsConsistentSolo(t *testing.T) {
	// nil profile.
	if isConsistentSolo(nil) {
		t.Error("nil profile should not be consistent solo")
	}
	// Too few bookings.
	if isConsistentSolo(&TravelProfile{Bookings: make([]Booking, 2)}) {
		t.Error("2 bookings should not be consistent solo")
	}
}

func TestHasLuggagePreference(t *testing.T) {
	// Not set.
	if hasLuggagePreference(&preferences.Preferences{}) {
		t.Error("should not have luggage pref when not configured")
	}
	// Set but no airports (not fully configured).
	if hasLuggagePreference(&preferences.Preferences{CarryOnOnly: true}) {
		t.Error("carry_on_only without airports should not count")
	}
	// Fully set.
	if !hasLuggagePreference(&preferences.Preferences{CarryOnOnly: true, HomeAirports: []string{"HEL"}}) {
		t.Error("should have luggage pref when carry_on_only + airports")
	}
}

func TestHasAccomPreference(t *testing.T) {
	if hasAccomPreference(&preferences.Preferences{}) {
		t.Error("should not have accom pref when empty")
	}
	if !hasAccomPreference(&preferences.Preferences{NoDormitories: true}) {
		t.Error("should have accom pref when NoDormitories set")
	}
	if !hasAccomPreference(&preferences.Preferences{MinHotelStars: 3}) {
		t.Error("should have accom pref when MinHotelStars set")
	}
}
