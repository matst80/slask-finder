package tracking

import (
	"encoding/json"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"
	"tornberg.me/facet-search/pkg/index"
)

type RabbitTrackingConfig struct {
	TrackingTopic string
	Url           string
}

type RabbitTracking struct {
	RabbitTrackingConfig
	connection *amqp.Connection
	channel    *amqp.Channel
}

func NewRabbitTracking(config RabbitTrackingConfig) *RabbitTracking {
	ret := RabbitTracking{
		RabbitTrackingConfig: config,
	}
	ret.Connect()
	return &ret
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
	t.channel = ch

	return nil
}

func (t *RabbitTracking) Close() error {
	defer t.connection.Close()
	return t.channel.Close()
}

func (t *RabbitTracking) send(data any) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return t.channel.Publish(
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
	SessionId uint32 `json:"session_id"`
	Event     uint16 `json:"event"`
}

type Session struct {
	*BaseEvent
	UserAgent    string `json:"user_agent,omitempty"`
	Ip           string `json:"ip,omitempty"`
	Language     string `json:"language,omitempty"`
	PragmaHeader string `json:"pragma,omitempty"`
}

func (rt *RabbitTracking) TrackSession(session_id uint32, r *http.Request) error {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-Ip")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}

	return rt.send(Session{
		BaseEvent:    &BaseEvent{Event: 0, SessionId: session_id},
		Language:     r.Header.Get("Accept-Language"),
		UserAgent:    r.UserAgent(),
		Ip:           ip,
		PragmaHeader: r.Header.Get("Pragma"),
	})
}

type Event struct {
	*BaseEvent
	Item     uint    `json:"item"`
	Position float32 `json:"position"`
}

type ImpressionEvent struct {
	*BaseEvent
	Items []Impression `json:"items"`
}

type CartEvent struct {
	*BaseEvent
	Item     uint `json:"item"`
	Quantity uint `json:"quantity"`
}

type SearchEventData struct {
	*index.Filters
	*BaseEvent
	Query string `json:"query"`
	Page  int    `json:"page"`
}

func (rt *RabbitTracking) TrackSearch(session_id uint32, filters *index.Filters, query string, page int) error {
	return rt.send(&SearchEventData{
		BaseEvent: &BaseEvent{Event: 1, SessionId: session_id},
		Filters:   filters,
		Query:     query,
		Page:      page,
	})

}

func (rt *RabbitTracking) TrackClick(session_id uint32, item_id uint, position float32) error {
	return rt.send(&Event{
		BaseEvent: &BaseEvent{Event: 2, SessionId: session_id},
		Item:      item_id,
		Position:  position,
	})
}

func (rt *RabbitTracking) TrackAddToCart(session_id uint32, item_id uint, quantity uint) error {
	return rt.send(&CartEvent{
		BaseEvent: &BaseEvent{Event: 3, SessionId: session_id},
		Item:      item_id,
		Quantity:  quantity,
	})
}

func (rt *RabbitTracking) TrackPurchase(session_id uint32, item_id uint, quantity uint) error {
	return rt.send(&CartEvent{
		BaseEvent: &BaseEvent{Event: 4, SessionId: session_id},
		Item:      item_id,
		Quantity:  quantity,
	})
}

func (rt *RabbitTracking) TrackImpressions(session_id uint32, viewedItems []Impression) error {
	return rt.send(&ImpressionEvent{
		BaseEvent: &BaseEvent{Event: 5, SessionId: session_id},
		Items:     viewedItems,
	})
}
