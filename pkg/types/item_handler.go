package types

import "sync"

// ItemHandler is an interface for handling items
type ItemHandler interface {
	HandleItem(item Item, wg *sync.WaitGroup)
}
