package stats

import (
	"time"

	"github.com/evanofslack/analogdb"
)

// posting is the count and mean popularity by UTC ISO weekday and hour
func (b *builder) posting() analogdb.StatsPosting {
	var sums [7][24]float64
	var counts [7][24]int
	for _, p := range b.posts {
		t := time.Unix(p.Time, 0).UTC()
		day := (int(t.Weekday()) + 6) % 7
		sums[day][t.Hour()] += p.Popularity
		counts[day][t.Hour()]++
	}
	cells := make([]analogdb.StatsPostingCell, 0, 7*24)
	for day := range 7 {
		for hour := range 24 {
			cell := analogdb.StatsPostingCell{Weekday: day + 1, Hour: hour, Count: counts[day][hour]}
			if cell.Count > 0 {
				cell.Popularity = pop(sums[day][hour] / float64(cell.Count))
			}
			cells = append(cells, cell)
		}
	}
	return analogdb.StatsPosting{Cells: cells}
}
