package tracking

import (
	"log"
	"net/http"

	"github.com/matst80/slask-finder/pkg/messaging"
	"github.com/matst80/slask-finder/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

// type RabbitTrackingConfig struct {
// 	TrackingTopic string
// 	Url           string
// }

type RabbitTracking struct {
	//RabbitTrackingConfig
	country    string
	connection *amqp.Connection
	//channel    *amqp.Channel
}

const trackingTopic = "tracking"

func NewRabbitTracking(url, country string) (*RabbitTracking, error) {
	ret := RabbitTracking{
		connection: nil,
		country:    country,
	}
	err := ret.connect(url)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (t *RabbitTracking) connect(url string) error {

	conn, err := amqp.Dial(url)
	if err != nil {
		return err
	}
	t.connection = conn
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	return messaging.DefineTopic(ch, "global", trackingTopic)

}

func (t *RabbitTracking) Close() error {
	return t.connection.Close()
}

func (t *RabbitTracking) send(data any) error {
	return messaging.SendChange(t.connection, "global", trackingTopic, data)
}

type BaseEvent struct {
	SessionId int    `json:"session_id"`
	Country   string `json:"country,omitempty"`
	Context   string `json:"context,omitempty"`
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
		BaseEvent:    &BaseEvent{Event: 0, SessionId: sessionId, Country: rt.country, Context: "b2c"},
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
		BaseEvent:       &BaseEvent{Event: 1, SessionId: sessionId, Country: rt.country, Context: "b2c"},
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

func (rt *RabbitTracking) TrackAction(sessionId int, value types.TrackingAction) error {
	return rt.send(&ActionEvent{
		BaseEvent: &BaseEvent{Event: 6, SessionId: sessionId, Country: rt.country, Context: "b2c"},
		Action:    value.Action,
		Reason:    value.Reason,
	})
}
