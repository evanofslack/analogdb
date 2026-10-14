package server

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/evanofslack/analogdb"
)

// wireCursor is the JSON form of a cursor before base64url encoding.
type wireCursor struct {
	Sort  string          `json:"s"`
	Value json.RawMessage `json:"v"`
	ID    int             `json:"id"`
	Seed  int             `json:"seed,omitempty"`
	Order string          `json:"o,omitempty"`
}

// encodeCursor builds the opaque cursor pointing after post.
func encodeCursor(sort analogdb.PostSort, asc bool, post *analogdb.Post, seed int) (string, error) {
	wire := wireCursor{Sort: sort.String(), ID: post.Id}
	wire.Order = cursorOrder(sort, asc)
	var value any
	switch sort {
	case analogdb.PostSortTime:
		value = post.Time
	case analogdb.PostSortScore:
		value = post.Score
	case analogdb.PostSortRandom:
		value = randomSortKey(post.Id, seed)
		wire.Seed = seed
	default:
		return "", fmt.Errorf("invalid sort parameter: %s", sort.String())
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	wire.Value = raw
	b, err := json.Marshal(wire)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// decodeCursor parses an opaque cursor for the given sort and direction.
// It returns the cursor and the seed it carries, or 0.
func decodeCursor(s string, sort analogdb.PostSort, asc bool) (*analogdb.Cursor, int, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, 0, badRequest("invalid cursor")
	}
	var wire wireCursor
	if err := json.Unmarshal(b, &wire); err != nil || len(wire.Value) == 0 {
		return nil, 0, badRequest("invalid cursor")
	}
	if wire.Sort != sort.String() || wire.Order != cursorOrder(sort, asc) {
		return nil, 0, badRequest("cursor does not match sort")
	}
	cursor := &analogdb.Cursor{ID: wire.ID}
	switch sort {
	case analogdb.PostSortTime, analogdb.PostSortScore:
		if err := json.Unmarshal(wire.Value, &cursor.Value); err != nil {
			return nil, 0, badRequest("invalid cursor")
		}
	case analogdb.PostSortRandom:
		if err := json.Unmarshal(wire.Value, &cursor.Hash); err != nil || !isRandomSortKey(cursor.Hash) {
			return nil, 0, badRequest("invalid cursor")
		}
		if !validSeed(wire.Seed) {
			return nil, 0, badRequest("invalid cursor")
		}
	default:
		return nil, 0, badRequest("invalid cursor")
	}
	return cursor, wire.Seed, nil
}

// cursorOrder is the order a cursor carries, empty for the default
func cursorOrder(sort analogdb.PostSort, asc bool) string {
	if asc && sort != analogdb.PostSortRandom {
		return analogdb.PostOrderAsc.String()
	}
	return ""
}

func validSeed(seed int) bool {
	return seed > 0 && seed <= math.MaxInt32
}

// randomSortKey matches md5(id::text || seed::text) in postgres.
func randomSortKey(id, seed int) string {
	sum := md5.Sum([]byte(strconv.Itoa(id) + strconv.Itoa(seed)))
	return hex.EncodeToString(sum[:])
}

func isRandomSortKey(s string) bool {
	if len(s) != md5.Size*2 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
