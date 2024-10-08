package sync

import (
	"log"

	"tornberg.me/facet-search/pkg/index"
)

type RabbitMasterChangeHandler struct {
	Master RabbitTransportMaster
}

func (r *RabbitMasterChangeHandler) ItemsUpserted(items []index.DataItem) {
	if len(items) == 0 {
		return
	}
	err := r.Master.ItemsUpserted(items)
	if err != nil {
		log.Printf("Failed to send item changed %v", err)
	}
	log.Printf("Items changed %d", len(items))
}

func (r *RabbitMasterChangeHandler) ItemDeleted(id uint) {

	err := r.Master.SendItemDeleted(id)
	if err != nil {
		log.Printf("Failed to send item deleted %v", err)
	}
	log.Printf("Item deleted %d", id)
}
