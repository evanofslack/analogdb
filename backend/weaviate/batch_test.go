package weaviate

import (
	"reflect"
	"testing"
)

func TestBatchBy(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	tests := []struct {
		name      string
		batchSize int
		expected  [][]int
	}{
		{"zero batch size", 0, [][]int{{1, 2, 3, 4, 5}}},
		{"negative batch size", -1, [][]int{{1, 2, 3, 4, 5}}},
		{"batch size one", 1, [][]int{{1}, {2}, {3}, {4}, {5}}},
		{"batch size n", 2, [][]int{{1, 2}, {3, 4}, {5}}},
		{"batch size equal to length", 5, [][]int{{1, 2, 3, 4, 5}}},
		{"batch size larger than length", 10, [][]int{{1, 2, 3, 4, 5}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := batchBy(items, tt.batchSize)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
