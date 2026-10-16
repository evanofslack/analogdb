package stats

import "math"

func round(x float64, places int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	p := math.Pow(10, float64(places))
	return math.Round(x*p) / p
}

func r3(x float64) float64 { return round(x, 3) }

// pop turns a 0 to 1 percent rank into the 0 to 100 popularity shown
func pop(x float64) float64 { return round(100*x, 1) }

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}
