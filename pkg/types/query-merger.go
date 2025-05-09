package types

import "sync"

type IQueryMerger interface {
	Add(func() *ItemList)
	Intersect(func() *ItemList)
	Exclude(func() *ItemList)
}

type QueryMerger struct {
	wg         *sync.WaitGroup
	MergeFirst bool
	isFirst    bool
	l          sync.Mutex
	merger     Merger
	result     *ItemList
	exclude    *ItemList
}

type Merger = func(current *ItemList, next *ItemList, isFirst bool)

func NewQueryMerger(result *ItemList) *QueryMerger {
	return &QueryMerger{
		wg:      &sync.WaitGroup{},
		isFirst: true,
		result:  result,
		merger: func(current *ItemList, next *ItemList, isFirst bool) {
			if isFirst {
				current.Merge(next)
			} else {
				current.Intersect(*next)
			}
		},
		exclude: &ItemList{},
	}
}

func NewCustomMerger(result *ItemList, merger Merger) *QueryMerger {
	return &QueryMerger{
		wg:      &sync.WaitGroup{},
		isFirst: true,
		result:  result,
		merger:  merger,
		exclude: &ItemList{},
	}
}

func (m *QueryMerger) Add(getResult func() *ItemList) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		items := getResult()
		if items == nil && m.isFirst {
			return
		}

		m.l.Lock()
		defer m.l.Unlock()

		m.merger(m.result, items, m.isFirst)
		m.isFirst = false

	}()
}

func (m *QueryMerger) Intersect(getResult func() *ItemList) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		items := getResult()

		m.l.Lock()
		defer m.l.Unlock()
		m.result.Intersect(*items)
	}()
}

func (m *QueryMerger) Exclude(getResult func() *ItemList) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		items := getResult()
		m.l.Lock()
		defer m.l.Unlock()
		m.exclude.Merge(items)
	}()
}

func (m *QueryMerger) Wait() {
	m.wg.Wait()
	m.result.Exclude(m.exclude)
}
