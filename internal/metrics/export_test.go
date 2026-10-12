package metrics

import (
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// AddSample runs the series bound on its own.
func AddSample(pts []model.UsageSample, s model.UsageSample, maxPoints int, age time.Duration) []model.UsageSample {
	b := &seriesBuf{pts: pts}
	b.add(s, maxPoints, age)
	return b.pts
}
