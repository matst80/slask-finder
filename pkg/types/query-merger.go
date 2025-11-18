package types

import (
	"context"
	"sync"
)

// IQueryMerger defines the public merging interface.
type IQueryMerger interface {
	Add(func() *ItemList)       // Add a constraint (first call seeds, subsequent calls intersect)
	Intersect(func() *ItemList) // Force intersection regardless of Add semantics
	Exclude(func() *ItemList)   // Collect items to be excluded after all additions/intersections
}

// Merger is a custom merging strategy hook.
type Merger = func(ctx context.Context, current *ItemList, next *ItemList, isFirst bool)

// QueryMerger coordinates concurrent set operations over ItemLists.
// Semantics (default constructor):
//
//	First Add with a non-nil result -> seed result with that set.
//	Subsequent Adds -> result = result ∩ next
//	Add with nil on first call -> treated as 'universe' (no-op).
//	Exclusions are accumulated and applied once in Wait().
type QueryMerger struct {
	ctx        context.Context
	wg         *sync.WaitGroup
	MergeFirst bool // retained for compatibility (not actively used)
	isFirst    bool
	l          sync.Mutex
	merger     Merger
	result     *ItemList
	exclude    *ItemList
}

// GetClone waits for completion and unions the internal result into output.
func (m *QueryMerger) GetClone(output *ItemList) {
	m.wg.Wait()
	if output == nil {
		return
	}
	output.Merge(m.result)
}

// NewQueryMerger builds a QueryMerger with default (seed + intersect) semantics.
func NewQueryMerger(ctx context.Context, result *ItemList) *QueryMerger {
	return &QueryMerger{
		ctx:     ctx,
		wg:      &sync.WaitGroup{},
		isFirst: true,
		result:  result,
		merger: func(ctx context.Context, current *ItemList, next *ItemList, isFirst bool) {
			if next == nil {
				// nil means "no restriction"
				return
			}
			if isFirst {
				current.Merge(next)
			} else {
				current.Intersect(next)
			}
		},
		exclude: &ItemList{},
	}
}

// NewCustomMerger allows providing a custom merge strategy.
func NewCustomMerger(ctx context.Context, result *ItemList, merger Merger) *QueryMerger {
	return &QueryMerger{
		ctx:     ctx,
		wg:      &sync.WaitGroup{},
		isFirst: true,
		result:  result,
		merger:  merger,
		exclude: &ItemList{},
	}
}

// Add applies the default seeded-intersection merge semantics.
func (m *QueryMerger) Add(getResult func(ctx context.Context) *ItemList) {
	m.wg.Go(func() {
		items := getResult(m.ctx)
		m.l.Lock()
		m.merger(m.ctx, m.result, items, m.isFirst)
		// Flip isFirst after first meaningful evaluation
		if m.isFirst && (items != nil) {
			m.isFirst = false
		} else if m.isFirst && items == nil {
			// nil does not seed but prevents repeated seeding attempts
			m.isFirst = false
		}
		m.l.Unlock()
	})
}

// Intersect forces an intersection with the provided set.
func (m *QueryMerger) Intersect(getResult func() *ItemList) {
	m.wg.Go(func() {
		items := getResult()
		if items == nil {
			return
		}
		m.l.Lock()
		m.result.Intersect(items)
		m.isFirst = false
		m.l.Unlock()
	})
}

// Exclude collects items to remove from the final result.
func (m *QueryMerger) Exclude(getResult func() *ItemList) {
	m.wg.Go(func() {
		items := getResult()
		if items == nil {
			return
		}
		m.l.Lock()
		m.exclude.Merge(items)
		m.l.Unlock()
	})
}

// Wait blocks until all operations complete and then applies exclusions.
func (m *QueryMerger) Wait() {
	m.wg.Wait()
	m.result.Exclude(m.exclude)
}
