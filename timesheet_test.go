package timesheet

import (
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	input := `# tuesday
2026-09-01 09:00-12:30 acme-corp: environment setup

2026-09-01 12:30-13:00 acme-corp: lunch prep
2026-09-01 13:00-17:30 acme-corp: implement parser
2026-09-02 09:00-17:00 internal: planning
`
	sheet, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sheet.Entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(sheet.Entries))
	}
	if got, want := sheet.TotalMinutes(), 990; got != want {
		t.Fatalf("total minutes = %d, want %d", got, want)
	}
}

func TestAdjacentEntriesDoNotOverlap(t *testing.T) {
	input := "2026-09-01 09:00-12:00 acme: a\n2026-09-01 12:00-13:00 acme: b\n"
	if _, err := Parse(strings.NewReader(input)); err != nil {
		t.Fatalf("back-to-back entries should be valid: %v", err)
	}
}

func TestParseInvalid(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"too few fields", "two words", "expected"},
		{"invalid calendar date", "2026-02-30 09:00-10:00 acme: work", "invalid date"},
		{"month out of range", "2026-13-01 09:00-10:00 acme: work", "invalid date"},
		{"zero duration", "2026-09-01 09:00-09:00 acme: work", "end time must be after start time"},
		{"hour out of range", "2026-09-01 24:00-25:00 acme: work", "invalid time"},
		{"time missing minutes", "2026-09-01 9-10 acme: work", "invalid time"},
		{"missing colon", "2026-09-01 09:00-10:00 acme work", `missing ":"`},
		{"empty project", "2026-09-01 09:00-10:00 : work", "empty project"},
		{"empty description", "2026-09-01 09:00-10:00 acme:   ", "empty description"},
		{"overlapping entries", "2026-09-01 09:00-12:00 acme: a\n2026-09-01 11:00-13:00 acme: b", "overlapping"},
		{"nested overlap", "2026-09-01 09:00-17:00 acme: a\n2026-09-01 10:00-10:30 acme: b", "overlapping"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(c.input))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), c.wantErr)
			}
		})
	}
}

func TestOvernightShift(t *testing.T) {
	input := "2026-09-01 22:00-06:00 acme: night shift\n"
	sheet, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := sheet.Entries[0].Minutes(), 8*60; got != want {
		t.Fatalf("Minutes() = %d, want %d", got, want)
	}
	if got, want := sheet.TotalMinutes(), 8*60; got != want {
		t.Fatalf("TotalMinutes() = %d, want %d", got, want)
	}
}

func TestOvernightShiftTouchingNextDayDoesNotOverlap(t *testing.T) {
	input := "2026-09-01 22:00-06:00 acme: night shift\n" +
		"2026-09-02 06:00-08:00 acme: morning handoff\n"
	if _, err := Parse(strings.NewReader(input)); err != nil {
		t.Fatalf("back-to-back overnight shift should be valid: %v", err)
	}
}

func TestOvernightShiftOverlapsNextDay(t *testing.T) {
	input := "2026-09-01 22:00-06:00 acme: night shift\n" +
		"2026-09-02 05:00-09:00 acme: morning shift\n"
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatalf("expected an overlap error")
	}
	if !strings.Contains(err.Error(), "overlapping") {
		t.Fatalf("error %q does not mention overlapping", err.Error())
	}
}

func TestFormat(t *testing.T) {
	input := "2026-09-01 09:00-12:30 acme-corp: environment setup\n" +
		"2026-09-01 13:00-17:00 acme-corp: implement parser\n"
	sheet, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "2026-09-01\n" +
		"  09:00-12:30  acme-corp  environment setup\n" +
		"  13:00-17:00  acme-corp  implement parser\n" +
		"  daily total  7h30m\n\n" +
		"total  7h30m\n"

	if got := Format(sheet); got != want {
		t.Fatalf("Format() =\n%q\nwant\n%q", got, want)
	}
}
