package ranking

import "sort"

// RankByValueCount export function
func RankByValueCount(src map[string]int) PairList {
pl := make(PairList, len(src))
i := 0
for k, v := range src {
	pl[i] = Pair{k, v}
	i++
}
sort.Sort(sort.Reverse(pl))
return pl
}

type Pair struct {
	Key string
	Value int
}
// PairList export type
type PairList []Pair

func (p PairList) Len() int { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int){ p[i], p[j] = p[j], p[i] }

