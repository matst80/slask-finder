package types

import (
	"iter"
)

// ItemHandler is an interface for handling items
type ItemHandler interface {
	HandleItem(item Item)
	HandleItems(itemIter iter.Seq[Item])
}
