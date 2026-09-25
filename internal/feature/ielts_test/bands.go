package ielts_test

import "math"

// IELTSOverall averages criterion bands the way IELTS reports an overall
// band: a mean ending in .25 rounds up to .5, one ending in .75 rounds up
// to the next whole band, anything else rounds down to the nearest half.
func IELTSOverall(bands []float64) float64 {
	if len(bands) == 0 {
		return 0
	}
	sum := 0.0
	for _, b := range bands {
		sum += b
	}
	mean := sum / float64(len(bands))
	whole := math.Floor(mean)
	switch frac := mean - whole; {
	case frac < 0.25:
		return whole
	case frac < 0.75:
		return whole + 0.5
	default:
		return whole + 1
	}
}
