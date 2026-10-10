package server

// propKind is the JSON type an event prop must have
type propKind int

const (
	propString propKind = iota
	propInt
	propNumber
	propBool
	propStringMap
)

type eventSchema struct {
	version int
	props   map[string]propKind
}

// eventSchemas is the allowlist of UI events. It mirrors EventMap in
// web/lib/analytics.ts
var eventSchemas = map[string]eventSchema{
	"page_view": {version: 1},
	"web_vital": {version: 1, props: map[string]propKind{
		"metric": propString,
		"value":  propNumber,
		"rating": propString,
	}},
	"search": {version: 1, props: map[string]propKind{
		"q":            propString,
		"q_norm":       propString,
		"mode":         propString,
		"source":       propString,
		"filters":      propStringMap,
		"result_count": propInt,
		"outcome":      propString,
		"page":         propInt,
	}},
	"search_result_click": {version: 1, props: map[string]propKind{
		"position":  propInt,
		"post_id":   propInt,
		"search_id": propString,
	}},
	"post_view": {version: 1, props: map[string]propKind{
		"from":      propString,
		"post_id":   propInt,
		"search_id": propString,
	}},
	"post_download": {version: 1, props: map[string]propKind{
		"post_id":   propInt,
		"search_id": propString,
	}},
	"outbound_click": {version: 1, props: map[string]propKind{
		"post_id": propInt,
		"target":  propString,
	}},
	"similar_click": {version: 1, props: map[string]propKind{
		"from_post_id": propInt,
		"post_id":      propInt,
		"position":     propInt,
	}},
}
