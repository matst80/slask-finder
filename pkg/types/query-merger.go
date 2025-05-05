package types

import "sync"

type QueryMerger struct {
	wg         *sync.WaitGroup
	MergeFirst bool
	isFirst    bool
	l          sync.Mutex
	result     *ItemList
	exclude    *ItemList
}

func NewQueryMerger(result *ItemList) *QueryMerger {
	wg := &sync.WaitGroup{}
	res := &QueryMerger{
		wg:         wg,
		MergeFirst: true,
		isFirst:    true,
		result:     result,
		exclude:    &ItemList{},
	}
	return res
}

func (m *QueryMerger) Add(getResult func() *ItemList) {
	m.wg.Add(1)
	go func() {
		items := getResult()
		defer m.wg.Done()
		m.l.Lock()
		defer m.l.Unlock()
		if m.MergeFirst && m.isFirst {
			m.isFirst = false
			m.result.Merge(items)
		} else {
			m.result.Intersect(*items)
		}
	}()
}

func (m *QueryMerger) Exclude(getResult func() *ItemList) {
	m.wg.Add(1)
	go func() {
		items := getResult()
		defer m.wg.Done()
		m.l.Lock()
		defer m.l.Unlock()
		m.exclude.Merge(items)
	}()
}

func (m *QueryMerger) Wait() {
	m.wg.Wait()
	m.result.Exclude(m.exclude)
}
