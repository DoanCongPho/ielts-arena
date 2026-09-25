package ielts_test

import "testing"

func TestIeltsOverall(t *testing.T) {
	cases := []struct {
		bands []float64
		want  float64
	}{
		{[]float64{6, 6, 6, 7}, 6.5}, // 6.25 rounds up to 6.5
		{[]float64{6, 7, 7, 7}, 7},   // 6.75 rounds up to 7
		{[]float64{6, 6, 7, 7}, 6.5}, // 6.5 stays
		{[]float64{6, 6, 6, 6}, 6},   // whole stays
		{[]float64{6, 6.5, 6, 6}, 6}, // 6.125 rounds down
		{[]float64{8, 9, 9, 9}, 9},   // 8.75 rounds up to 9
		{[]float64{4, 5, 5, 5}, 5},   // 4.75
		{[]float64{5, 5, 5, 6}, 5.5}, // 5.25
	}
	for _, c := range cases {
		if got := IELTSOverall(c.bands); got != c.want {
			t.Errorf("ieltsOverall(%v) = %v, want %v", c.bands, got, c.want)
		}
	}
}
