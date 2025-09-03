package tracking

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/matst80/slask-finder/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitTrackingConfig struct {
	TrackingTopic string
	Url           string
}

type RabbitTracking struct {
	RabbitTrackingConfig
	connection *amqp.Connection
	//channel    *amqp.Channel
}

func NewRabbitTracking(config RabbitTrackingConfig) (*RabbitTracking, error) {
	ret := RabbitTracking{
		RabbitTrackingConfig: config,
	}
	err := ret.Connect()
	return &ret, err
}

func (t *RabbitTracking) Connect() error {

	conn, err := amqp.Dial(t.Url)
	if err != nil {
		return err
	}
	t.connection = conn
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if err := ch.ExchangeDeclare(
		t.TrackingTopic, // name
		"topic",         // type
		true,            // durable
		false,           // auto-delete
		false,           // internal
		false,           // noWait
		nil,             // arguments
	); err != nil {
		return err
	}

	if _, err = ch.QueueDeclare(
		t.TrackingTopic, // name of the queue
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // noWait
		nil,             // arguments
	); err != nil {
		return err
	}

	return nil
}

func (t *RabbitTracking) Close() error {
	return t.connection.Close()
}

func (t *RabbitTracking) send(data any) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	ch, err := t.connection.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	return ch.Publish(
		t.TrackingTopic,
		t.TrackingTopic,
		true,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        bytes,
		},
	)
}

type BaseEvent struct {
	SessionId int    `json:"session_id"`
	Event     uint16 `json:"event"`
}

type Session struct {
	*BaseEvent
	UserAgent    string `json:"user_agent,omitempty"`
	Ip           string `json:"ip,omitempty"`
	Language     string `json:"language,omitempty"`
	PragmaHeader string `json:"pragma,omitempty"`
}

func (rt *RabbitTracking) TrackSession(sessionId int, r *http.Request) {
	ip := r.Header.Get("X-Real-Ip")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}

	err := rt.send(Session{
		BaseEvent:    &BaseEvent{Event: 0, SessionId: sessionId},
		Language:     r.Header.Get("Accept-Language"),
		UserAgent:    r.UserAgent(),
		Ip:           ip,
		PragmaHeader: r.Header.Get("Pragma"),
	})
	if err != nil {
		log.Println("Error sending session event: ", err)
	}
}

type Event struct {
	*BaseEvent
	Item     uint    `json:"item"`
	Position float32 `json:"position"`
}

type ActionEvent struct {
	*BaseEvent
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type SearchEventData struct {
	*types.Filters
	*BaseEvent
	NumberOfResults int    `json:"noi"`
	Query           string `json:"query"`
	Page            int    `json:"page"`
	Referer         string `json:"referer"`
}

func (rt *RabbitTracking) TrackSearch(sessionId int, filters *types.Filters, resultLen int, query string, page int, r *http.Request) {
	referer := r.Header.Get("Referer")
	err := rt.send(&SearchEventData{
		BaseEvent:       &BaseEvent{Event: 1, SessionId: sessionId},
		Filters:         filters,
		Query:           query,
		NumberOfResults: resultLen,
		Page:            page,
		Referer:         referer,
	})
	if err != nil {
		log.Println("Error sending search event: ", err)
	}

}

func (rt *RabbitTracking) TrackAction(sessionId int, value TrackingAction) error {
	return rt.send(&ActionEvent{
		BaseEvent: &BaseEvent{Event: 6, SessionId: sessionId},
		Action:    value.Action,
		Reason:    value.Reason,
	})
}
