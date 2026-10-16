package stats

import "math"

func dot(a, b []float32) float64 {
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

func norm(a []float32) float64 {
	return math.Sqrt(dot(a, a))
}

// unit returns a normalized copy, or nil for a zero vector
func unit(a []float32) []float32 {
	n := norm(a)
	if n == 0 {
		return nil
	}
	out := make([]float32, len(a))
	for i, x := range a {
		out[i] = float32(float64(x) / n)
	}
	return out
}

func cosine(a, b []float32) float64 {
	na, nb := norm(a), norm(b)
	if na == 0 || nb == 0 {
		return 0
	}
	return dot(a, b) / (na * nb)
}

// meanVector averages vectors, nil when there are none
func meanVector(vecs [][]float32) []float32 {
	if len(vecs) == 0 {
		return nil
	}
	sum := make([]float64, len(vecs[0]))
	for _, v := range vecs {
		for i, x := range v {
			sum[i] += float64(x)
		}
	}
	out := make([]float32, len(sum))
	for i, x := range sum {
		out[i] = float32(x / float64(len(vecs)))
	}
	return out
}

func sub(a, b []float32) []float32 {
	out := make([]float32, len(a))
	for i := range a {
		out[i] = a[i] - b[i]
	}
	return out
}

// pca2 projects the rows onto their first two principal components. It works
// on the n by n Gram matrix with power iteration, so it is cheap for few rows
// of many dimensions. Results are deterministic for a given row order.
func pca2(rows [][]float32) (xs, ys []float64) {
	n := len(rows)
	if n == 0 {
		return nil, nil
	}
	centered := make([][]float32, n)
	center := meanVector(rows)
	for i, r := range rows {
		centered[i] = sub(r, center)
	}
	gram := make([][]float64, n)
	for i := range gram {
		gram[i] = make([]float64, n)
		for j := range gram[i] {
			gram[i][j] = dot(centered[i], centered[j])
		}
	}
	xs, lambda := powerIteration(gram)
	for i := range gram {
		for j := range gram[i] {
			gram[i][j] -= lambda * xs[i] * xs[j]
		}
	}
	ys, _ = powerIteration(gram)
	return xs, ys
}

// powerIteration returns the unit top eigenvector of a symmetric matrix and its eigenvalue
func powerIteration(m [][]float64) ([]float64, float64) {
	n := len(m)
	v := make([]float64, n)
	for i := range v {
		v[i] = 1 + float64(i)/float64(n)
	}
	normalize(v)
	var lambda float64
	for range pcaIterations {
		next := make([]float64, n)
		for i := range m {
			for j, x := range m[i] {
				next[i] += x * v[j]
			}
		}
		lambda = normalize(next)
		if lambda == 0 {
			return v, 0
		}
		v = next
	}
	return v, lambda
}

func normalize(v []float64) float64 {
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	n := math.Sqrt(sum)
	if n == 0 {
		return 0
	}
	for i := range v {
		v[i] /= n
	}
	return n
}
