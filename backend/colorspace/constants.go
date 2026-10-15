package colorspace

// Tuning values. All lightness and chroma values are OKLCH, hues are degrees.
const (
	// NeutralChroma is the chroma below which a color is black, gray or white
	NeutralChroma = 0.04

	// DarkMaxL and DarkMaxChroma make tinted near-black shadows neutral
	DarkMaxL      = 0.25
	DarkMaxChroma = 0.08

	// NavyMinChroma and RedMinChroma keep blue-gray shadows out of navy and
	// send muted brick reds to brown
	NavyMinChroma = 0.06
	RedMinChroma  = 0.08

	// BlackMaxL and WhiteMinL split neutrals by lightness
	BlackMaxL = 0.35
	WhiteMinL = 0.9

	// MatchRadius is the OKLab distance within which a cluster matches a hex query
	MatchRadius = 0.09

	// HexThreshold is the default minimum share for a hex query
	HexThreshold = 0.15

	// RainbowHueOffset rotates the start of each rainbow loop to red
	RainbowHueOffset = 15.0

	// RainbowCycles is how many hue loops the rainbow is split into
	RainbowCycles = 128

	// RainbowHash spreads post ids over the rainbow cycles
	RainbowHash = 2654435761
)

// Warm modifiers turn dark warm hues brown, pale warm hues tan and dark yellows olive
const (
	BrownHueMin = 35.0
	BrownHueMax = 95.0
	BrownMaxL   = 0.6
	// BrownMaxChroma keeps saturated burnt oranges orange unless darker than BrownDarkL
	BrownMaxChroma = 0.1
	BrownDarkL     = 0.55
	TanMaxChroma   = 0.1
	OliveHueMin    = 95.0
	OliveHueMax    = 135.0
	OliveMaxL      = 0.7
	OliveLowC      = 0.08
	OliveLowCMax   = 0.85
)

// Pink takes light reds and mid to light magentas
const (
	PinkHueMin        = 340.0
	PinkHueMax        = 25.0
	PinkMinL          = 0.7
	PinkMagentaHueMin = 320.0
	PinkMagentaHueMax = 350.0
	PinkMagentaMinL   = 0.55
)

// hueRanges assigns the remaining chromatic colors by hue. A range starts at
// its From and runs to the next range, red wraps around 360.
var hueRanges = []struct {
	From float64
	Name string
}{
	{0, Red},
	{40, Orange},
	{75, Yellow},
	{115, Green},
	{165, Teal},
	{235, Navy},
	{290, Purple},
	{350, Red},
}

// BucketThresholds are the default minimum shares for a bucket query
var BucketThresholds = map[string]float64{
	Red:    0.2,
	Orange: 0.2,
	Tan:    0.3,
	Yellow: 0.15,
	Green:  0.25,
	Olive:  0.3,
	Teal:   0.25,
	Navy:   0.25,
	Purple: 0.15,
	Pink:   0.2,
	Gray:   0.6,
	Black:  0.5,
	White:  0.4,
	Brown:  0.35,
}
