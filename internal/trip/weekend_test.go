package trip

import (
	"testing"
)

func TestWeekendOptions_Defaults(t *testing.T) {
	opts := WeekendOptions{}
	opts.defaults()

	if opts.Nights != 2 {
		t.Errorf("Nights = %d, want 2", opts.Nights)
	}
}

func TestWeekendOptions_DefaultsPreserve(t *testing.T) {
	opts := WeekendOptions{Nights: 3}
	opts.defaults()

	if opts.Nights != 3 {
		t.Errorf("Nights = %d, want 3", opts.Nights)
	}
}

func TestWeekendOptions_DefaultsNegativeNights(t *testing.T) {
	opts := WeekendOptions{Nights: -1}
	opts.defaults()
	if opts.Nights != 2 {
		t.Errorf("Nights = %d, want 2 for negative input", opts.Nights)
	}
}

func TestParseMonth_LongFormat(t *testing.T) {
	depart, ret, display, err := parseMonth("July-2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if display != "July 2026" {
		t.Errorf("display = %q, want July 2026", display)
	}
	// First Friday of July 2026 is July 3.
	if depart != "2026-07-03" {
		t.Errorf("depart = %q, want 2026-07-03", depart)
	}
	if ret != "2026-07-05" {
		t.Errorf("return = %q, want 2026-07-05", ret)
	}
}

func TestParseMonth_ShortFormat(t *testing.T) {
	_, _, display, err := parseMonth("2026-08")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if display != "August 2026" {
		t.Errorf("display = %q, want August 2026", display)
	}
}

func TestParseMonth_Invalid(t *testing.T) {
	_, _, _, err := parseMonth("not-a-month")
	if err == nil {
		t.Error("expected error for invalid month")
	}
}

func TestParseMonth_LowercaseLong(t *testing.T) {
	depart, ret, display, err := parseMonth("july-2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if display != "July 2026" {
		t.Errorf("display = %q, want July 2026", display)
	}
	if depart != "2026-07-03" {
		t.Errorf("depart = %q, want 2026-07-03", depart)
	}
	if ret != "2026-07-05" {
		t.Errorf("return = %q, want 2026-07-05", ret)
	}
}

func TestParseMonth_ShortName(t *testing.T) {
	_, _, display, err := parseMonth("Jan-2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if display != "January 2026" {
		t.Errorf("display = %q, want January 2026", display)
	}
}

func TestParseMonth_LowercaseShortName(t *testing.T) {
	_, _, display, err := parseMonth("jan-2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if display != "January 2026" {
		t.Errorf("display = %q, want January 2026", display)
	}
}

func TestParseMonth_FirstFridayWhenMonthStartsOnFriday(t *testing.T) {
	// January 2027 starts on a Friday.
	depart, ret, _, err := parseMonth("2027-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if depart != "2027-01-01" {
		t.Errorf("depart = %q, want 2027-01-01 (Jan 1 2027 is Friday)", depart)
	}
	if ret != "2027-01-03" {
		t.Errorf("return = %q, want 2027-01-03", ret)
	}
}

func TestParseMonth_FirstFridayWhenMonthStartsOnSaturday(t *testing.T) {
	// May 2027 starts on Saturday, so first Friday is May 7.
	depart, _, _, err := parseMonth("2027-05")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if depart != "2027-05-07" {
		t.Errorf("depart = %q, want 2027-05-07 (first Friday after Sat May 1)", depart)
	}
}

func TestFindWeekendGetaways_EmptyOrigin(t *testing.T) {
	_, err := FindWeekendGetaways(t.Context(), "", WeekendOptions{Month: "july-2026"})
	if err == nil {
		t.Error("expected error for empty origin")
	}
}

func TestFindWeekendGetaways_InvalidMonth(t *testing.T) {
	_, err := FindWeekendGetaways(t.Context(), "HEL", WeekendOptions{Month: "invalid"})
	if err == nil {
		t.Error("expected error for invalid month")
	}
}
