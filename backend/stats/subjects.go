package stats

import "github.com/evanofslack/analogdb"

// subjects is the lift of each subject for films with enough posts, null
// where the film has too few posts with the subject
func (b *builder) subjects() analogdb.StatsSubjects {
	overall := make([]int, len(Subjects))
	for _, p := range b.posts {
		for i, w := range Subjects {
			if p.words[w] {
				overall[i]++
			}
		}
	}
	total := float64(len(b.posts))

	rows := []analogdb.StatsSubjectRow{}
	for _, f := range b.films {
		if len(f.posts) < MinHeatmapPosts {
			continue
		}
		row := analogdb.StatsSubjectRow{Slug: f.slug, Lift: make([]*float64, len(Subjects)), Count: make([]int, len(Subjects))}
		for _, p := range f.posts {
			for i, w := range Subjects {
				if p.words[w] {
					row.Count[i]++
				}
			}
		}
		for i := range Subjects {
			if row.Count[i] < MinLiftCount || overall[i] == 0 {
				continue
			}
			lift := r3((float64(row.Count[i]) / float64(len(f.posts))) / (float64(overall[i]) / total))
			row.Lift[i] = &lift
		}
		rows = append(rows, row)
	}
	return analogdb.StatsSubjects{Words: Subjects, Rows: rows}
}
