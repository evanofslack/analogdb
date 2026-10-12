package analogdb

import "testing"

func TestParseImageHash(t *testing.T) {
	for _, s := range []string{"0000000000000000", "c3a1f0e0d8b8b0f0", "ffffffffffffffff", "7fffffffffffffff"} {
		h, err := ParseImageHash(s)
		if err != nil {
			t.Fatalf("%s: %v", s, err)
		}
		if got := FormatImageHash(h); got != s {
			t.Errorf("want %s round trip, got %s", s, got)
		}
	}
	if h, _ := ParseImageHash("ffffffffffffffff"); h != -1 {
		t.Errorf("want top bit kept as -1, got %d", h)
	}
	if h, _ := ParseImageHash("C3A1F0E0D8B8B0F0"); FormatImageHash(h) != "c3a1f0e0d8b8b0f0" {
		t.Error("want upper case accepted")
	}
	for _, s := range []string{"", "abc", "c3a1f0e0d8b8b0f", "c3a1f0e0d8b8b0f00", "g3a1f0e0d8b8b0f0", "-3a1f0e0d8b8b0f0"} {
		if _, err := ParseImageHash(s); ErrorCode(err) != ERRBADREQUEST {
			t.Errorf("%q: want bad request, got %v", s, err)
		}
	}
}
