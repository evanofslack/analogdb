package analogdb

import "testing"

func TestSlug(t *testing.T) {
	tests := []struct {
		parts []string
		want  string
	}{
		{[]string{"kodak", "portra 400"}, "kodak-portra-400"},
		{[]string{"voigtländer", "bessa r2"}, "voigtlander-bessa-r2"},
		{[]string{"hasselblad", "500c/m"}, "hasselblad-500c-m"},
		{[]string{"rollei", "rolleiflex 2.8f"}, "rollei-rolleiflex-2-8f"},
		{[]string{"Canon", "AE-1"}, "canon-ae-1"},
		{[]string{" ilford ", "hp5+"}, "ilford-hp5"},
		{[]string{""}, ""},
	}
	for _, tt := range tests {
		if got := Slug(tt.parts...); got != tt.want {
			t.Errorf("Slug(%q) = %q, want %q", tt.parts, got, tt.want)
		}
	}
}
