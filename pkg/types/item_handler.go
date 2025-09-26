package types

import (
	"iter"
)

// ItemHandler is an interface for handling items
type ItemHandler interface {
	HandleItems(itemIter iter.Seq[Item])
}
