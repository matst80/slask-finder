package sync

import (
	"log"

	"github.com/matst80/slask-finder/pkg/types"
)

type RabbitMasterChangeHandler struct {
	Master RabbitTransportMaster
}

func (r *RabbitMasterChangeHandler) ItemsUpserted(items []types.Item) {
	if len(items) == 0 {
		log.Fatalln("No items to upsert")
		return
	}
	err := r.Master.ItemsUpserted(items)
	if err != nil {
		log.Printf("Failed to send item changed %v", err)
	}
	log.Printf("Items changed %d", len(items))
}

func (r *RabbitMasterChangeHandler) PriceLowered(items []types.Item) {

	err := r.Master.SendPriceLowered(items)
	if err != nil {
		log.Printf("Failed to send price updates %v", err)
	}
	log.Printf("Items with price lowered %d", len(items))
}

func (r *RabbitMasterChangeHandler) ItemDeleted(id uint) {

	err := r.Master.SendItemDeleted(id)
	if err != nil {
		log.Printf("Failed to send item deleted %v", err)
	}
	log.Printf("Item deleted %d", id)
}
