package sorting

// import (
// 	"github.com/matst80/slask-finder/pkg/types"
// )

// // FacetSortingCoordinator handles the coordination between facet and sorting functionality
// // This utility extracts the overlapping logic that requires both facet and sorting data
// type FacetSortingCoordinator struct {
// 	facetHandler   *FacetItemHandler
// 	sortingHandler *SortingItemHandler
// }

// func NewFacetSortingCoordinator(facetHandler *FacetItemHandler, sortingHandler *SortingItemHandler) *FacetSortingCoordinator {
// 	return &FacetSortingCoordinator{
// 		facetHandler:   facetHandler,
// 		sortingHandler: sortingHandler,
// 	}
// }

// // GetFieldSortData extracts field sorting information from facets
// // This replaces the direct access to idx.Facets in the sorting makeFieldSort method
// func (coord *FacetSortingCoordinator) GetFieldSortData(overrides SortOverride) (map[uint]types.Facet, SortOverride) {
// 	if coord.facetHandler == nil {
// 		return nil, overrides
// 	}

// 	return coord.facetHandler.Facets, overrides
// }

// // UpdateFieldSortWithFacets updates the sorting field sort based on current facets
// // This method can be called when facets change to update the corresponding sort
// func (coord *FacetSortingCoordinator) UpdateFieldSortWithFacets(idx *FacetItemHandler) {
// 	if coord.sortingHandler != nil && coord.sortingHandler.Sorting != nil {
// 		// Use empty overrides for now - this can be enhanced later
// 		overrides := SortOverride{}

// 		// Update field sort using facet data
// 		coord.sortingHandler.Sorting.makeFieldSort(idx, overrides)
// 	}
// }

// // GetSortingForFacets extracts sorting information for facet display
// // This can be used to get sorting data when displaying facets in UI
// func (coord *FacetSortingCoordinator) GetSortingForFacets() *types.ByValue {
// 	if coord.sortingHandler != nil && coord.sortingHandler.Sorting != nil {
// 		return coord.sortingHandler.Sorting.FieldSort
// 	}
// 	return nil
// }

// // Note: This utility file isolates the coordination logic between facets and sorting
// // Future enhancements can be added here to further separate concerns while maintaining
// // the necessary integration points between the two systems
