package kinds

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// SetClock fixes the clock the Cron Jobs view computes Next execution from, for the tests of
// package kinds_test (the goldens); it returns a function that restores the previous clock.
func SetClock(now func() time.Time) (restore func()) {
	old := clock
	clock = now
	return func() { clock = old }
}

func cronJob(schedule, zone string, status map[string]any) map[string]any {
	spec := map[string]any{"schedule": schedule}
	if zone != "" {
		spec["timeZone"] = zone
	}
	return map[string]any{
		"metadata": map[string]any{"name": "c", "namespace": "ns", "creationTimestamp": "2026-10-11T08:10:00Z"},
		"spec":     spec,
		"status":   status,
	}
}

// withZone sets spec.timeZone, even to "".
func withZone(obj map[string]any, zone string) map[string]any {
	obj["spec"].(map[string]any)["timeZone"] = zone
	return obj
}

func utc(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestCronJobNext(t *testing.T) {
	tests := []struct {
		name    string
		obj     map[string]any
		now     string
		want    string // RFC 3339; "" for no next run
		problem string
	}{
		{name: "every two minutes", obj: cronJob("*/2 * * * *", "", nil), now: "2026-10-11T12:01:30Z", want: "2026-10-11T12:02:00Z"},
		{name: "strictly after now", obj: cronJob("*/2 * * * *", "", nil), now: "2026-10-11T12:02:00Z", want: "2026-10-11T12:04:00Z"},
		{name: "no time zone is UTC", obj: cronJob("0 9 * * *", "", nil), now: "2026-10-11T12:00:00Z", want: "2026-10-12T09:00:00Z"},
		{name: "empty time zone is UTC", obj: withZone(cronJob("0 9 * * *", "", nil), ""), now: "2026-10-11T08:00:00Z", want: "2026-10-11T09:00:00Z"},
		{name: "descriptor", obj: cronJob("@yearly", "", nil), now: "2026-10-11T12:00:00Z", want: "2027-01-01T00:00:00Z"},
		{name: "weekday names", obj: cronJob("0 9 * * mon-fri", "", nil), now: "2026-10-10T12:00:00Z", want: "2026-10-12T09:00:00Z"}, // a Saturday
		{
			name: "time zone across the end of summer time",
			obj:  cronJob("30 3 * * *", "Europe/Berlin", nil), now: "2026-10-24T12:00:00Z",
			want: "2026-10-25T02:30:00Z", // 03:30 CET, an hour after summer time ended
		},
		{name: "time zone west of UTC", obj: cronJob("0 9 * * *", "America/New_York", nil), now: "2026-10-11T12:00:00Z", want: "2026-10-11T13:00:00Z"},
		{name: "zone in the schedule", obj: cronJob("TZ=Asia/Tokyo 0 9 * * *", "", nil), now: "2026-10-11T12:00:00Z", want: "2026-10-12T00:00:00Z"},
		{name: "zone in the schedule beats timeZone", obj: cronJob("CRON_TZ=Asia/Tokyo 0 9 * * *", "Europe/Berlin", nil), now: "2026-10-11T12:00:00Z", want: "2026-10-12T00:00:00Z"},
		{name: "zone prefix without a space", obj: cronJob("TZ=UTC", "", nil), now: "2026-10-11T12:00:00Z", problem: cronBadSchedule},
		{name: "unknown zone in the schedule", obj: cronJob("TZ=Mars/Base 0 9 * * *", "", nil), now: "2026-10-11T12:00:00Z", problem: cronBadSchedule},
		{name: "unknown time zone", obj: cronJob("0 9 * * *", "Mars/Base", nil), now: "2026-10-11T12:00:00Z", problem: cronBadZone},
		{name: "Local is refused", obj: cronJob("0 9 * * *", "local", nil), now: "2026-10-11T12:00:00Z", problem: cronBadZone},
		{name: "path in the time zone", obj: cronJob("0 9 * * *", "../../etc/passwd", nil), now: "2026-10-11T12:00:00Z", problem: cronBadZone},
		{name: "no schedule", obj: cronJob("", "", nil), now: "2026-10-11T12:00:00Z", problem: cronBadSchedule},
		{name: "out of range", obj: cronJob("0 24 * * *", "", nil), now: "2026-10-11T12:00:00Z", problem: cronBadSchedule},
		{name: "six fields", obj: cronJob("0 0 9 * * *", "", nil), now: "2026-10-11T12:00:00Z", problem: cronBadSchedule},
		{name: "a date that never comes", obj: cronJob("0 0 30 2 *", "", nil), now: "2026-10-11T12:00:00Z"},
		{name: "a field that allows nothing", obj: cronJob(", * * * *", "", nil), now: "2026-10-11T12:00:00Z"},
		{
			name: "every counts from the last schedule",
			obj:  cronJob("@every 90m", "", map[string]any{"lastScheduleTime": "2026-10-11T11:10:00Z"}), now: "2026-10-11T12:00:00Z",
			want: "2026-10-11T12:40:00Z",
		},
		{
			name: "every counts from the creation before the first run",
			obj:  cronJob("@every 90m", "", nil), now: "2026-10-11T12:00:00Z",
			want: "2026-10-11T12:40:00Z", // 08:10, 09:40, 11:10, 12:40
		},
		{
			name: "every with missed runs stays on its grid",
			obj:  cronJob("@every 1h", "", map[string]any{"lastScheduleTime": "2026-10-10T23:15:00Z"}), now: "2026-10-11T12:00:00Z",
			want: "2026-10-11T12:15:00Z",
		},
		{
			name: "every from a future creation",
			obj:  cronJob("@every 1h", "", nil), now: "2026-10-11T07:00:00Z",
			want: "2026-10-11T09:10:00Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, problem := cronJobNext(tt.obj, utc(tt.now))
			var want time.Time
			if tt.want != "" {
				want = utc(tt.want)
			}
			if !got.Equal(want) || problem != tt.problem {
				t.Errorf("got %v %q, want %v %q", got.UTC(), problem, want, tt.problem)
			}
		})
	}
}

// TestCronJobNextStaleness pins the documented staleness: Next execution is computed when the row
// is built and rendered relative to the moment the table draws.
func TestCronJobNextStaleness(t *testing.T) {
	defer SetClock(func() time.Time { return utc("2026-10-11T12:01:30Z") })()
	i := cronJobs.ColumnIndex("nextExecution")
	text := func(obj map[string]any, at string) string {
		row := model.BuildRow(cronJobs, "test", &unstructured.Unstructured{Object: obj})
		return model.CellText(cronJobs.Columns[i], row.Cells[i], utc(at))
	}

	running := cronJob("*/2 * * * *", "", nil)
	if got := text(running, "2026-10-11T12:01:45Z"); got != "In less than a minute" {
		t.Errorf("before the run: %q", got)
	}
	// The run comes due; until the controller records it, the row built at 12:01:30 reads "now".
	if got := text(running, "2026-10-11T12:02:00Z"); got != "now" {
		t.Errorf("when the run is due: %q", got)
	}
	// The controller records the run, the object changes, and the rebuilt row shows the next one.
	SetClock(func() time.Time { return utc("2026-10-11T12:02:01Z") })
	running["status"] = map[string]any{"lastScheduleTime": "2026-10-11T12:02:00Z"}
	if got := text(running, "2026-10-11T12:02:10Z"); got != "In about 1 min" {
		t.Errorf("after the controller's update: %q", got)
	}

	// A suspended cron job's status does not change when its schedule comes due, so a row built
	// before stays at "now" until something else changes the object.
	SetClock(func() time.Time { return utc("2026-10-11T12:01:30Z") })
	suspended := cronJob("*/2 * * * *", "", nil)
	suspended["spec"].(map[string]any)["suspend"] = true
	if got := text(suspended, "2026-10-11T12:30:00Z"); got != "now" {
		t.Errorf("suspended, long after: %q", got)
	}
}

func TestCronJobHealth(t *testing.T) {
	tests := []struct {
		name   string
		status map[string]any
		want   model.Status
	}{
		{"never ran", nil, model.StatusOK},
		{"last run succeeded", map[string]any{"lastScheduleTime": "2026-10-11T11:00:00Z", "lastSuccessfulTime": "2026-10-11T11:00:09Z"}, model.StatusOK},
		{"last run failed", map[string]any{"lastScheduleTime": "2026-10-11T11:00:00Z", "lastSuccessfulTime": "2026-10-11T10:00:09Z"}, model.StatusWarning},
		{"never succeeded", map[string]any{"lastScheduleTime": "2026-10-11T11:00:00Z"}, model.StatusWarning},
		{"running", map[string]any{"lastScheduleTime": "2026-10-11T11:00:00Z", "active": []any{map[string]any{"name": "j"}}}, model.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := cronJob("0 * * * *", "", tt.status)
			cell, status := cronJobHealth(obj, "")
			wantCell := tt.want // the icon has the row's colour; an ok row has no icon
			if tt.want == model.StatusOK {
				wantCell = model.StatusNone
			}
			if status != tt.want || cell.Status != wantCell {
				t.Errorf("got %q (cell %+v), want %q", status, cell, tt.want)
			}
		})
	}
	// A schedule problem outranks the rest, suspended or not.
	obj := cronJob("bad", "", map[string]any{"lastScheduleTime": "2026-10-11T11:00:00Z"})
	obj["spec"].(map[string]any)["suspend"] = true
	if cell, status := cronJobHealth(obj, cronBadSchedule); status != model.StatusError || cell.Text != cronBadSchedule {
		t.Errorf("problem: %q %+v", status, cell)
	}
}

func TestTimeZoneCache(t *testing.T) {
	a, ok := timeZone("Europe/Berlin")
	b, ok2 := timeZone("Europe/Berlin")
	if !ok || !ok2 || a != b {
		t.Errorf("timeZone = %v %v, %v %v; want the same cached zone twice", a, ok, b, ok2)
	}
	if _, ok := timeZone("Nowhere/Never"); ok {
		t.Error("an unknown zone loaded")
	}
	if _, cached := zones.Load("Nowhere/Never"); cached {
		t.Error("an unknown zone was cached")
	}
	if _, ok := timeZone("LOCAL"); ok {
		t.Error("Local loaded")
	}
}

func TestEveryNextBounds(t *testing.T) {
	now := utc("2026-10-11T12:00:00Z")
	if got := everyNext(0, now, now); !got.IsZero() {
		t.Errorf("zero delay = %v", got)
	}
	// A base so far back that the elapsed time saturates: no panic, no wrapped time.
	if got := everyNext(time.Duration(1<<62), time.Time{}.Add(time.Hour), now); !got.IsZero() && got.Before(time.Time{}) {
		t.Errorf("saturated = %v", got)
	}
	if got := everyNext(time.Second, now.Add(-1500*time.Millisecond), now); !got.Equal(now.Add(time.Second)) {
		t.Errorf("sub-second base = %v", got) // the base truncates to 11:59:58, so the grid is on whole seconds
	}
}
