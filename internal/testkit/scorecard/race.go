//go:build race

package scorecard

// raceEnabled says whether the test binary runs under the race detector: its measurements are
// recorded, but the budgets are judged without it.
const raceEnabled = true
