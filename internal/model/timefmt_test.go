package model

import (
	"slices"
	"strings"
	"testing"
	"time"
	_ "time/tzdata" // Europe/Berlin and America/New_York on any test machine
)

// Layouts of a few regions, as internal/locale converts them from CLDR (its tests check that).
var (
	germanLayout = TimeLayout{Date: "02.01.2006", Time: "15:04:05"}
	usLayout     = TimeLayout{Date: "1/2/2006", Time: "3:04:05 PM"}
)

func berlin(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// TestTimeFormatZoneAndSummerTime: the owner's example, a certificate's Not Before shown as
// "2026-09-10T17:41:01+02:00", reads "10.09.2026 17:41:01" in Germany; Berlin's summer time is
// UTC+2 and its winter time UTC+1, which the zone database decides, not Cogpit.
func TestTimeFormatZoneAndSummerTime(t *testing.T) {
	de := germanLayout.In(berlin(t))
	summer := time.Date(2026, 9, 10, 15, 41, 1, 0, time.UTC)
	winter := time.Date(2026, 12, 10, 15, 41, 1, 0, time.UTC)
	for _, tc := range []struct{ got, want string }{
		{de.Format(summer), "10.09.2026 17:41:01"},
		{de.Format(winter), "10.12.2026 16:41:01"},
		{de.FormatDate(summer), "10.09.2026"},
		{de.FormatTime(winter), "16:41:01"},
		{de.FormatClock(summer), "17:41"},
		{de.FormatDay(summer), "10.09."},
		{de.FormatPrecise(summer.Add(123456789)), "10.09.2026 17:41:01.123"},
		{TimeFormat{}.Format(summer), "2026-09-10 15:41:01"}, // the zero value: ISO 8601 in UTC
		{TimeFormat{}.FormatDay(summer), "09-10"},
		{TimeFormat{}.Format(time.Time{}), ""},
	} {
		if tc.got != tc.want {
			t.Errorf("got %q, want %q", tc.got, tc.want)
		}
	}
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	us := usLayout.In(ny)
	for _, tc := range []struct{ got, want string }{
		{us.Format(summer), "9/10/2026 11:41:01 AM"},
		{us.FormatClock(winter), "10:41 AM"},
		{us.FormatDay(summer), "9/10"},
		{us.FormatPrecise(winter), "12/10/2026 10:41:01.000 AM"},
	} {
		if tc.got != tc.want {
			t.Errorf("got %q, want %q", tc.got, tc.want)
		}
	}
}

// TestDerivedLayouts: the time without seconds and the date without the year, for the layouts of
// several regions and for unusual ones.
func TestDerivedLayouts(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"15:04:05", "15:04"},
		{"3:04:05 PM", "3:04 PM"},
		{"PM 3:04:05", "PM 3:04"},
		{"15.04.05", "15.04"},
		{"15:04:05.000", "15:04"},
		{"15 h 04 min 05 s", "15 h 04 min"},
		{"15:04", "15:04"},
	} {
		if got := clockLayout(tc.in); got != tc.want {
			t.Errorf("clockLayout(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	for _, tc := range []struct{ in, want string }{
		{"02.01.2006", "02.01."},
		{"2.1.2006", "2.1."},
		{"1/2/2006", "1/2"},
		{"02/01/2006", "02/01"},
		{"02-01-2006", "02-01"},
		{"2006-01-02", "01-02"},
		{"2006/1/2", "1/2"},
		{"2006. 01. 02.", "01. 02."},
		{"Monday, 2. January 2006", "Monday, 2. January"},
		{"02.01", "02.01"},
	} {
		if got := dayLayout(tc.in); got != tc.want {
			t.Errorf("dayLayout(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	for _, tc := range []struct{ in, want string }{
		{"15:04:05", "15:04:05.000"},
		{"3:04:05 PM", "3:04:05.000 PM"},
		{"15:04:05.000", "15:04:05.000"},
		{"15:04", "15:04"},
	} {
		if got := preciseLayout(tc.in); got != tc.want {
			t.Errorf("preciseLayout(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if got, want := LayoutElements("Mon, 02.01.2006 3:04:05.000 PM -07:00 MST"), []string{"weekday", "day", "month",
		"year", "hour", "minute", "second", "fraction", "ampm", "zone", "zone"}; !slices.Equal(got, want) {
		t.Errorf("LayoutElements = %v, want %v", got, want)
	}
	if got := germanLayout.DateTimeLayouts(); !slices.Equal(got, []string{"02.01.2006 15:04:05", "02.01.2006 15:04", "02.01.2006"}) {
		t.Errorf("DateTimeLayouts = %q", got)
	}
	if got := usLayout.TimeOfDayLayouts(); !slices.Equal(got, []string{"3:04:05 PM", "3:04 PM"}) {
		t.Errorf("TimeOfDayLayouts = %q", got)
	}
}

// TestMarksWrittenInTheFormat: a kind writes an absolute time as a mark, which the drawer writes in
// the user's zone and format: in values, chips and tables, a date alone where marked so, and
// without touching the sections it was given.
func TestMarksWrittenInTheFormat(t *testing.T) {
	at := time.Date(2026, 9, 10, 15, 41, 1, 0, time.UTC)
	de := germanLayout.In(berlin(t))
	if got := de.WriteMarks("Running, started " + MarkTime(at)); got != "Running, started 10.09.2026 17:41:01" {
		t.Errorf("WriteMarks = %q", got)
	}
	if got := de.WriteMarks(MarkDate(at) + " / " + MarkTime(at.Add(time.Hour))); got != "10.09.2026 / 10.09.2026 18:41:01" {
		t.Errorf("WriteMarks of two = %q", got)
	}
	if MarkTime(time.Time{}) != "" || MarkDate(time.Time{}) != "" || HasMarks("plain") {
		t.Error("the zero time has a mark, or plain text holds one")
	}
	// What is no mark of Cogpit's is shown as it came: an anchor without a time, one without its end.
	for _, s := range []string{"a\uFFF9b\uFFFBc", "x\uFFF9", "\uFFF92026-13-99\uFFFB"} {
		if got := de.WriteMarks(s); got != s {
			t.Errorf("WriteMarks(%q) = %q", s, got)
		}
	}

	sections := []DetailSection{
		{Title: "Plain", Items: []DetailItem{{Name: "Name", Value: "web"}}},
		{Title: "Times", Items: []DetailItem{
			{Name: "Created", Value: "5m ago (" + MarkTime(at) + ")"},
			{Name: "Conditions", Value: "Ready", Chips: []Chip{{Text: "Ready since " + MarkTime(at)}}},
		}, Table: CellTable([]Column{{ID: "v", Title: "Version"}, {ID: "c", Title: "Created"}},
			[][]Cell{{{Text: "1.0"}, {Text: MarkDate(at)}}})},
	}
	before := sections[1].Items[0].Value
	got := de.WriteSections(sections)
	if &got[0].Items[0] != &sections[0].Items[0] {
		t.Error("a section without marks was copied")
	}
	if sections[1].Items[0].Value != before || !HasMarks(sections[1].Table.Rows[0][1]) {
		t.Error("WriteSections changed its input")
	}
	if v := got[1].Items[0].Value; v != "5m ago (10.09.2026 17:41:01)" {
		t.Errorf("Created = %q", v)
	}
	if c := got[1].Items[1].Chips[0].Text; c != "Ready since 10.09.2026 17:41:01" {
		t.Errorf("chip = %q", c)
	}
	if r, c := got[1].Table.Rows[0][1], got[1].Table.Cells[0][1].Text; r != "10.09.2026" || c != r {
		t.Errorf("table cell = %q, %q", r, c)
	}
	for _, sec := range got {
		for _, it := range sec.Items {
			if HasMarks(it.Value) || strings.ContainsRune(it.Value, '\uFFFB') {
				t.Errorf("a mark is left in %q", it.Value)
			}
		}
	}
}

// TestColumnTime: an absolute time in a list is written when it is drawn, in the format and zone
// given, ISO 8601 in now's zone without one; it sorts by its timestamp.
func TestColumnTime(t *testing.T) {
	col := Column{ID: "lastSeen", Title: "Last Seen", Type: ColumnTime}
	at := time.Date(2026, 9, 10, 15, 41, 1, 0, time.UTC)
	cell := Cell{Num: at.Unix()}
	if got := CellText(col, cell, now); got != "2026-09-10 15:41:01" {
		t.Errorf("CellText = %q", got)
	}
	if got := germanLayout.In(berlin(t)).CellText(col, cell, now); got != "10.09.2026 17:41:01" {
		t.Errorf("TimeFormat.CellText = %q", got)
	}
	if got := germanLayout.In(berlin(t)).CellText(col, Cell{}, now); got != "" {
		t.Errorf("an empty time cell reads %q", got)
	}
	age := Column{ID: "age", Title: "Age", Type: ColumnAge}
	if got := germanLayout.In(berlin(t)).CellText(age, Cell{Num: now.Add(-5 * time.Minute).Unix()}, now); got != "5m" {
		t.Errorf("an age through a format reads %q", got)
	}
}
