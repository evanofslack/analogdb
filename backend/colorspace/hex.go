package colorspace

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidHex = errors.New("invalid hex color")

// ParseHex parses a 6 digit hex color, with or without a leading #
func ParseHex(s string) (RGB, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return RGB{}, ErrInvalidHex
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return RGB{}, ErrInvalidHex
	}
	return RGB{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v)}, nil
}

// IsHex reports whether s is a 6 digit hex color, with or without a leading #
func IsHex(s string) bool {
	_, err := ParseHex(s)
	return err == nil
}

// Hex formats the color as 6 lowercase hex digits without a #
func (c RGB) Hex() string {
	return fmt.Sprintf("%02x%02x%02x", c.R, c.G, c.B)
}

// Snap moves each channel to the nearest multiple of 17
func (c RGB) Snap() RGB {
	snap := func(v uint8) uint8 { return uint8((int(v) + 8) / 17 * 17) }
	return RGB{R: snap(c.R), G: snap(c.G), B: snap(c.B)}
}

// NormalizeHex parses a hex color and returns it snapped, lowercase and without a #
func NormalizeHex(s string) (string, error) {
	c, err := ParseHex(s)
	if err != nil {
		return "", err
	}
	return c.Snap().Hex(), nil
}

// NormalizeQuery turns a color query value into a bucket name or a normalized hex
func NormalizeQuery(s string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(s))
	if IsBucket(name) {
		return name, nil
	}
	hex, err := NormalizeHex(name)
	if err != nil {
		return "", fmt.Errorf("unknown color %q", s)
	}
	return hex, nil
}

// DefaultThreshold is the minimum share for a query value when n colors are queried
func DefaultThreshold(value string, n int) float64 {
	threshold, ok := BucketThresholds[value]
	if !ok {
		threshold = HexThreshold
	}
	if n > 1 {
		threshold /= float64(n)
	}
	return threshold
}
