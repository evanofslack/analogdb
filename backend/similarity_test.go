package analogdb

import (
	"slices"
	"testing"
)

func TestOrderPostsByIDsRepeatedIDs(t *testing.T) {
	posts := []*Post{{Id: 1}, {Id: 2}, {Id: 3}}
	ids := []int{3, 1, 3, 4, 1, 2, 3}

	got := OrderPostsByIDs(posts, ids)

	gotIDs := []int{}
	for _, p := range got {
		gotIDs = append(gotIDs, p.Id)
	}
	if want := []int{3, 1, 2}; !slices.Equal(gotIDs, want) {
		t.Errorf("ordered ids = %v, want %v", gotIDs, want)
	}
}

func TestDedupeIDs(t *testing.T) {
	got := DedupeIDs([]int{5, 2, 5, 5, 7, 2})
	if want := []int{5, 2, 7}; !slices.Equal(got, want) {
		t.Errorf("deduped ids = %v, want %v", got, want)
	}
}
