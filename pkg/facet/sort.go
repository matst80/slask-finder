package facet

import (
	"log"
	"strconv"
	"strings"
)

type SortIndex []uint

func (s *SortIndex) GetScore(id uint) float64 {
	for i, v := range *s {
		if v == id {
			return float64(i)
		}
	}
	return -1
}

func (s *SortIndex) Add(id uint) {
	*s = append(*s, id)
}

func (s *SortIndex) Remove(id uint) {
	for i, v := range *s {
		if v == id {
			*s = append((*s)[:i], (*s)[i+1:]...)
			return
		}
	}
}

func (s *SortIndex) FromString(data string) error {
	for _, str := range strings.Split(data, ",") {
		i, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		s.Add(uint(i))
	}
	return nil
}

func (s *SortIndex) ToString() string {
	var buffer strings.Builder
	for i, id := range *s {
		buffer.WriteString(strconv.Itoa(int(id)))
		if i != len(*s)-1 {
			buffer.WriteString(",")
		}
	}
	return buffer.String()
}

func (s *SortIndex) SortMapWithStaticPositions(ids IdList, staticPositions map[int]uint, breakAt int) []uint {
	if s == nil {
		log.Printf("SortIndex is nil")
		return []uint{}
	}

	l := min(len(ids), breakAt)
	sortedIds := make([]uint, l)
	idx := 0

	for _, id := range *s {
		if idx >= l {
			break
		}
		if sp, ok := staticPositions[idx]; ok {
			_, ok := ids[sp]
			if ok {
				sortedIds[idx] = sp
				idx++
			}
		}

		_, ok := ids[id]
		if ok {
			sortedIds[idx] = id

			idx++

		}
	}

	return sortedIds
}

func (s *SortIndex) SortMap(ids IdList, breakAt int) []uint {

	if s == nil {
		log.Printf("SortIndex is nil")
		return []uint{}
	}

	l := min(len(ids), breakAt)
	sortedIds := make([]uint, l)
	idx := 0

	for _, id := range *s {
		if idx >= l {
			break
		}
		_, ok := ids[id]
		if ok {
			sortedIds[idx] = id

			idx++

		}
	}

	return sortedIds

}

func (s *SortIndex) SortMatch(ids MatchList, breakAt int) []uint {

	if s == nil {
		log.Printf("SortIndex is nil")
		return []uint{}
	}

	l := min(len(ids), breakAt)
	sortedIds := make([]uint, l)
	idx := 0

	for _, id := range *s {
		if idx >= l {
			break
		}
		_, ok := ids[id]
		if ok {
			sortedIds[idx] = id

			idx++

		}
	}

	return sortedIds

}

type Lookup struct {
	Id    uint
	Value float64
}

type ByValue []Lookup

func (a ByValue) Len() int           { return len(a) }
func (a ByValue) Less(i, j int) bool { return a[i].Value < a[j].Value }
func (a ByValue) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
