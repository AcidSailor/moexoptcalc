package moexoptcalc_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/acidsailor/moexoptcalc"
)

// Date (un)marshals as a bare "2006-01-02" string, so a value round-trips to
// the exact wire form.
func TestDate_MarshalRoundTrip(t *testing.T) {
	const wire = `"2026-05-12"`
	var d moexoptcalc.Date
	if err := json.Unmarshal([]byte(wire), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := d.Format("2006-01-02"); got != "2026-05-12" {
		t.Errorf("calendar day = %q, want 2026-05-12", got)
	}
	out, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != wire {
		t.Errorf("round-trip = %s, want %s", out, wire)
	}
}

// A JSON null leaves the zero value in place — callers use *Date, so null is
// normally absence.
func TestDate_UnmarshalNull(t *testing.T) {
	var d moexoptcalc.Date
	if err := json.Unmarshal([]byte(`null`), &d); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if !d.IsZero() {
		t.Errorf("null should leave zero value, got %v", d)
	}
}

// An empty string must be a parse error, not a silent 0001-01-01 — the guard
// for required value-typed Date fields in responses.
func TestDate_UnmarshalEmptyStringErrors(t *testing.T) {
	var d moexoptcalc.Date
	if err := json.Unmarshal([]byte(`""`), &d); err == nil {
		t.Errorf("empty string should be a parse error, got zero value %v", d)
	}
}

// A non-date string is rejected with a descriptive error.
func TestDate_UnmarshalBadInputErrors(t *testing.T) {
	var d moexoptcalc.Date
	if err := json.Unmarshal([]byte(`"not-a-date"`), &d); err == nil {
		t.Error("malformed date should be a parse error, got nil")
	}
}

// NewDate is the blessed Go-side constructor; the wire day is the time's
// calendar day in its own location.
func TestNewDate_MarshalsCalendarDay(t *testing.T) {
	d := moexoptcalc.NewDate(time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC))
	out, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `"2026-05-12"` {
		t.Errorf("NewDate marshal = %s, want \"2026-05-12\"", out)
	}
}

// An empty string is likewise rejected for DateTime rather than silently zeroed.
func TestDateTime_UnmarshalEmptyStringErrors(t *testing.T) {
	var dt moexoptcalc.DateTime
	if err := json.Unmarshal([]byte(`""`), &dt); err == nil {
		t.Errorf("empty string should be a parse error, got zero value %v", dt)
	}
}

// NewDateTime normalizes to the Moscow zone, so the instant is preserved and
// MarshalJSON emits the Moscow wall-clock.
func TestNewDateTime_NormalizesToMoscow(t *testing.T) {
	// 18:39:22 UTC is 21:39:22 in Moscow (+3).
	dt := moexoptcalc.NewDateTime(
		time.Date(2026, 5, 12, 18, 39, 22, 0, time.UTC),
	)
	out, err := json.Marshal(dt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `"2026-05-12T21:39:22"` {
		t.Errorf("NewDateTime marshal = %s, want \"2026-05-12T21:39:22\"", out)
	}
}
