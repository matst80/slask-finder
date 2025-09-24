package types

type ItemHandler interface {
	HandleItem(item Item)
	HandleItems(items []Item)
	HandleItemUnsafe(item Item)
	Lock()
	Unlock()
}
