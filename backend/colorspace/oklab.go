// Package colorspace converts sRGB colors to OKLab and OKLCH and names them.
package colorspace

import "math"

// Lab is a color in OKLab
type Lab struct {
	L float64
	A float64
	B float64
}

// LCh is a color in OKLCH, with the hue in degrees from 0 to 360
type LCh struct {
	L float64
	C float64
	H float64
}

// RGB is an 8 bit sRGB color
type RGB struct {
	R uint8
	G uint8
	B uint8
}

func toLinear(c uint8) float64 {
	v := float64(c) / 255
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func fromLinear(v float64) uint8 {
	if v <= 0.0031308 {
		v *= 12.92
	} else {
		v = 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return uint8(math.Round(math.Max(0, math.Min(1, v)) * 255))
}

// ToLab converts sRGB to OKLab
func (c RGB) ToLab() Lab {
	r, g, b := toLinear(c.R), toLinear(c.G), toLinear(c.B)

	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)

	return Lab{
		L: 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		A: 1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		B: 0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}

// ToRGB converts OKLab to sRGB, clipping out of gamut values
func (c Lab) ToRGB() RGB {
	l := c.L + 0.3963377774*c.A + 0.2158037573*c.B
	m := c.L - 0.1055613458*c.A - 0.0638541728*c.B
	s := c.L - 0.0894841775*c.A - 1.2914855480*c.B
	l, m, s = l*l*l, m*m*m, s*s*s

	return RGB{
		R: fromLinear(4.0767416621*l - 3.3077115913*m + 0.2309699292*s),
		G: fromLinear(-1.2684380046*l + 2.6097574011*m - 0.3413193965*s),
		B: fromLinear(-0.0041960863*l - 0.7034186147*m + 1.7076147010*s),
	}
}

// ToLCh converts OKLab to OKLCH
func (c Lab) ToLCh() LCh {
	h := math.Atan2(c.B, c.A) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return LCh{L: c.L, C: math.Hypot(c.A, c.B), H: h}
}

// ToLab converts OKLCH to OKLab
func (c LCh) ToLab() Lab {
	rad := c.H * math.Pi / 180
	return Lab{L: c.L, A: c.C * math.Cos(rad), B: c.C * math.Sin(rad)}
}

// DeltaE is the Euclidean distance between two OKLab colors
func DeltaE(x, y Lab) float64 {
	dl, da, db := x.L-y.L, x.A-y.A, x.B-y.B
	return math.Sqrt(dl*dl + da*da + db*db)
}
