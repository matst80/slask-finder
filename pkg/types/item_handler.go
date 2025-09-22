package types

type ItemHandler interface {
	HandleItem(item Item) error
	HandleItems(items []Item) error
	HandleItemUnsafe(item Item) error
	Lock()
	Unlock()
}
