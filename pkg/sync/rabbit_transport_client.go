package sync

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// type RabbitTransportClient struct {
// 	RabbitConfig

// 	ClientName string
// 	//handler    index.UpdateHandler
// 	connection *amqp.Connection
// 	channel    *amqp.Channel
// 	quit       chan bool
// }

func DeclareBindAndConsume(ch *amqp.Channel, prefix string, topic ChangeTopic) (<-chan amqp.Delivery, error) {
	name := getName(prefix, topic)
	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}
	err = ch.QueueBind(q.Name, name, name, false, nil)
	if err != nil {
		return nil, err
	}
	return ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
}

// func (t *RabbitTransportClient) Connect(handlers ...types.ItemHandler) error {
// 	conn, err := amqp.DialConfig(t.Url, amqp.Config{
// 		Vhost:      t.VHost,
// 		Properties: amqp.NewConnectionProperties(),
// 	})
// 	//conn.Config.Vhost = t.VHost
// 	t.quit = make(chan bool)
// 	if err != nil {
// 		return err
// 	}
// 	t.connection = conn
// 	ch, err := conn.Channel()
// 	if err != nil {
// 		return err
// 	}
// 	//t.handler = handler
// 	t.channel = ch
// 	toAdd, err := t.declareBindAndConsume(t.ItemsUpsertedTopic)
// 	if err != nil {
// 		return err
// 	}
// 	log.Printf("Connected to rabbit upsert topic: %s", t.ItemsUpsertedTopic)
// 	go func(msgs <-chan amqp.Delivery) {
// 		for d := range msgs {

// 			var items []index.DataItem
// 			if err := json.Unmarshal(d.Body, &items); err == nil {
// 				log.Printf("Got upserts %d", len(items))
// 				for _, handler := range handlers {
// 					for _, item := range items {
// 						handler.HandleItem(&item)
// 					}
// 				}
// 			} else {
// 				log.Printf("Failed to unmarshal upset message %v", err)
// 			}
// 		}
// 	}(toAdd)

// 	// toDelete, err := t.declareBindAndConsume(t.ItemDeletedTopic)
// 	// if err != nil {
// 	// 	return err
// 	// }
// 	// log.Printf("Connected to rabbit delete topic: %s", t.ItemDeletedTopic)
// 	// go func(msgs <-chan amqp.Delivery) {
// 	// 	for d := range msgs {
// 	// 		var item uint
// 	// 		if err := json.Unmarshal(d.Body, &item); err == nil {
// 	// 			t.handler.DeleteItem(item)
// 	// 		}
// 	// 	}
// 	// }(toDelete)

// 	fieldUpdates, err := t.declareBindAndConsume(t.FieldChangeTopic)
// 	if err != nil {
// 		return err
// 	}
// 	log.Printf("Connected to rabbit field topic: %s", t.FieldChangeTopic)
// 	go func(msgs <-chan amqp.Delivery) {
// 		for d := range msgs {
// 			var changes []types.FieldChange
// 			if err := json.Unmarshal(d.Body, &changes); err == nil {
// 				log.Printf("Got field changes %d, change implementation when it works", len(changes))
// 				//t.handler.UpdateFields(changes)
// 			} else {
// 				log.Printf("Failed to unmarshal field change message %v", err)
// 			}
// 		}
// 	}(fieldUpdates)
// 	return nil
// }

// func (t *RabbitTransportClient) Close() {
// 	if (t.channel != nil) && (!t.channel.IsClosed()) {
// 		t.channel.Close()
// 	}
// 	if (t.connection != nil) && (!t.connection.IsClosed()) {
// 		t.connection.Close()
// 	}
// 	//t.quit <- true

// }
