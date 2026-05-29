package forecast

import (
	"math"

	"github.com/hd-health/hd-health/internal/domain"
)

func DaysToThreshold(v domain.Volume, growthBytesPerDay float64, thresholdPercent float64) *int {
	if v.Capacity <= 0 {
		return nil
	}
	targetUsed := int64(float64(v.Capacity) * thresholdPercent / 100)
	remaining := targetUsed - v.Used
	if remaining <= 0 {
		d := 0
		return &d
	}
	if growthBytesPerDay <= 0 {
		return nil
	}
	days := int(math.Ceil(float64(remaining) / growthBytesPerDay))
	return &days
}

func EnrichVolume(v domain.Volume, growth float64) domain.VolumeReport {
	r := domain.VolumeReport{Volume: v, GrowthBytesPerDay: growth}
	r.ForecastDaysTo85 = DaysToThreshold(v, growth, 85)
	r.ForecastDaysTo95 = DaysToThreshold(v, growth, 95)
	return r
}
