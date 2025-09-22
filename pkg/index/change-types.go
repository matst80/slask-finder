package index

import "github.com/matst80/slask-finder/pkg/types"

type ItemChangeHandler interface {
	ItemsChanged(item []types.Item)
}

type ItemUpdateHandler interface {
	HandleItems(item []types.Item)
}

type FieldChangeHandler interface {
	FieldsChanged(item []types.FieldChange)
}

type UpdateHandler interface {
	UpdateFields(changes []types.FieldChange)
}
