package v24

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// TestAdsetScheduleOmitempty verifies that StartTime/EndTime are only serialized when set.
// They are pointers so `omitempty` actually fires: a nil schedule must be omitted rather than
// marshaled as the zero "0001-01-01T00:00:00+0000", which Meta rejects on an already-started
// ad set ("The start_time cannot be edited if the Ad Set has already started").
func TestAdsetScheduleOmitempty(t *testing.T) {
	b, err := json.Marshal(Adset{ID: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "start_time") || strings.Contains(string(b), "end_time") {
		t.Fatalf("nil schedule must be omitted, got: %s", b)
	}

	start := fb.Time(time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC))
	end := fb.Time(time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC))
	b, err = json.Marshal(Adset{ID: "1", StartTime: &start, EndTime: &end})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"start_time":"2026-07-21`) || !strings.Contains(string(b), `"end_time":"2026-07-28`) {
		t.Fatalf("set schedule must be serialized, got: %s", b)
	}

	// Round-trips back into a pointer.
	var out Adset
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.StartTime == nil || !time.Time(*out.StartTime).Equal(time.Time(start)) {
		t.Fatalf("StartTime did not round-trip: %+v", out.StartTime)
	}
}
