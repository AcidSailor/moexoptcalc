package moexoptcalc_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/acidsailor/moexoptcalc"
)

// The wire form is a bare "2006-01-02T15:04:05" with no zone, but MOEX
// timestamps are Moscow-local. DateTime must parse them through Europe/Moscow
// so the instant is correct — a naive UTC parse would be off by the offset.
func TestDateTime_UnmarshalAppliesMoscowZone(t *testing.T) {
	var dt moexoptcalc.DateTime
	if err := json.Unmarshal([]byte(`"2026-05-12T21:39:22"`), &dt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Wall-clock fields are the bare wire value.
	if got := dt.Format("2006-01-02T15:04:05"); got != "2026-05-12T21:39:22" {
		t.Errorf("wall clock = %q, want 2026-05-12T21:39:22", got)
	}

	// Crux: the instant is Moscow-local (UTC+3), not naive UTC. Moscow has had
	// a fixed +3 offset with no DST since 2014.
	if _, offset := dt.Zone(); offset != 3*60*60 {
		t.Errorf("zone offset = %ds, want %ds (Moscow +3)", offset, 3*60*60)
	}
	wantUTC := time.Date(2026, 5, 12, 18, 39, 22, 0, time.UTC)
	if !dt.UTC().Equal(wantUTC) {
		t.Errorf("UTC instant = %v, want %v", dt.UTC(), wantUTC)
	}
}

// MarshalJSON renders the Moscow wall-clock so the wire string round-trips
// to the exact value MOEX sent.
func TestDateTime_MarshalRoundTrip(t *testing.T) {
	const wire = `"2026-05-12T21:39:22"`
	var dt moexoptcalc.DateTime
	if err := json.Unmarshal([]byte(wire), &dt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(dt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != wire {
		t.Errorf("round-trip = %s, want %s", out, wire)
	}
}

// A JSON null leaves the zero value in place — callers use *DateTime, so
// null is normally absence (mirrors Date).
func TestDateTime_UnmarshalNull(t *testing.T) {
	var dt moexoptcalc.DateTime
	if err := json.Unmarshal([]byte(`null`), &dt); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if !dt.IsZero() {
		t.Errorf("null should leave zero value, got %v", dt)
	}
}
