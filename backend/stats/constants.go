package stats

import "time"

// LoadSince is how far back posts are loaded, the window start is then
// found from the data
var LoadSince = time.Unix(0, 0).UTC()

const (
	// MinMonthPosts is the fewest posts a month needs to be in the window. The
	// window starts at the first month after which every complete month has this many.
	MinMonthPosts = 200

	// ArrivalsAfterMonths is how long after the window start a first post has
	// to be for a film or camera to count as new. Earlier ones may only look
	// new because the window starts.
	ArrivalsAfterMonths = 18

	// Movers compare the last MoverRecentMonths complete months with the
	// MoverBaseMonths before them
	MoverRecentMonths = 12
	MoverBaseMonths   = 24

	SnapshotVersion = 1

	MinFilmPosts    = 80
	MinHeatmapPosts = 100
	MinCameraPosts  = 100
	MinTrendPosts   = 15
	MinBWPosts      = 30
	MinLiftCount    = 15
	MinGroupPosts   = 150
	AuthorCap       = 5

	ChromaThreshold = 0.05
	HueBins         = 24

	TypicalPosts  = 6
	ThemePosts    = 4
	SimilarFilms  = 3
	TrendTop      = 20
	BrandTop      = 8
	MoverCount    = 5
	MoverMinShare = 1.0
	ArrivalsTop   = 8

	PaletteSize       = 5
	PaletteIterations = 20

	pcaIterations = 500
)

// Subjects are the heatmap columns in display order
var Subjects = []string{
	"neon", "night", "city", "double exposure", "portrait", "dress", "couple", "room", "mountain",
	"lake", "autumn", "snow", "desert", "beach", "sunset", "flower", "cat", "street",
}

type Family struct {
	Key   string
	Name  string
	Films []string
}

const familyBW = "bw"

// Families are hand labeled from the centroid similarities. Films not listed
// join the family of their nearest listed film, B&W films join bw.
var Families = []Family{
	{Key: "landscape", Name: "Landscape and slide", Films: []string{
		"kodak-ektar-100", "fujifilm-fujichrome-velvia-50", "fujifilm-fujichrome-velvia-100",
		"fujifilm-fujichrome-provia-100", "kodak-ektachrome-e100",
	}},
	{Key: "portrait", Name: "Portrait", Films: []string{
		"kodak-portra-160", "kodak-portra-400", "fujifilm-fujicolor-pro-400h",
	}},
	{Key: "consumer", Name: "Everyday", Films: []string{
		"kodak-gold-200", "kodak-colorplus-200", "kodak-ultramax-400", "fujifilm-200", "fujifilm-400",
		"fujifilm-fujicolor-superia-x-tra-400", "fujifilm-fujicolor-c200", "kodak-pro-image-100",
	}},
	{Key: "night", Name: "Night and tungsten", Films: []string{
		"cinestill-800t", "kodak-vision3-500t", "lomography-800", "kodak-portra-800", "cinestill-400d",
		"kodak-vision3-250d", "cinestill-50d",
	}},
	{Key: familyBW, Name: "Black and white"},
}

// positionAnchor is the film whose families map position is kept non-negative
// on both axes, so the map doesn't mirror between builds
const positionAnchor = "kodak-portra-400"
