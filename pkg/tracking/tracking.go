package tracking

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"tornberg.me/facet-search/pkg/index"
)

type Tracking interface {
	TrackSession(session_id uint32, r *http.Request) error
	TrackSearch(session_id uint32, filters *index.Filters, query string) error
	TrackClick(session_id uint32, item_id uint, position float32) error
	TrackAddToCart(session_id uint32, item_id uint, quantity uint) error
	TrackPurchase(session_id uint32, item_id uint, quantity uint) error
	TrackImpression(session_id uint32, item_id uint, position float32) error
}

type ClickHouse struct {
	Conn driver.Conn
}

func NewClickHouse(addr string) (*ClickHouse, error) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{addr},
			// Auth: clickhouse.Auth{
			// 	Database: "default",
			// 	Username: "default",
			// 	Password: "<DEFAULT_USER_PASSWORD>",
			// },
			ClientInfo: clickhouse.ClientInfo{
				Products: []struct {
					Name    string
					Version string
				}{
					{Name: "slask-finder", Version: "0.1"},
				},
			},

			Debugf: func(format string, v ...interface{}) {
				fmt.Printf(format, v)
			},
			// TLS: &tls.Config{
			// 	InsecureSkipVerify: true,
			// },
		})
	)

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}
	return &ClickHouse{Conn: conn}, nil
}

func (ch *ClickHouse) Close() error {
	return ch.Conn.Close()
}

func (ch *ClickHouse) Query() {
	ctx := context.Background()
	rows, err := ch.Conn.Query(ctx, "select session_id,evt from user_action")
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var session_id uint32
		var evt uint16

		if err := rows.Scan(
			&session_id,
			&evt,
		); err != nil {
			log.Fatal(err)
		}
		log.Printf("session_id: %d, evt: %d",
			session_id, evt)
	}
}

const SearchEvent = 1
const ClickEvent = 2
const AddToCartEvent = 3
const PurchaseEvent = 4
const ImpressionEvent = 5

func (ch *ClickHouse) TrackSearch(session_id uint32, filters *index.Filters, query string) error {
	ctx := context.Background()
	facets := map[uint]string{}
	for _, filter := range filters.StringFilter {
		facets[filter.Id] = filter.Value
	}

	for _, filter := range filters.IntegerFilter {
		facets[filter.Id] = fmt.Sprintf("%d-%d", filter.Min, filter.Max)
	}

	for _, filter := range filters.NumberFilter {
		facets[filter.Id] = fmt.Sprintf("%f-%f", filter.Min, filter.Max)
	}
	return ch.Conn.Exec(ctx, "INSERT INTO user_search (session_id, evt, query, facets, timestamp) VALUES (?, ?, ?, ?,?)", session_id, SearchEvent, query, facets, time.Now())

}

func (ch *ClickHouse) TrackClick(session_id uint32, item_id uint, position float32) error {
	return ch.TrackEvent(session_id, PurchaseEvent, item_id, position/10.0)
}

func (ch *ClickHouse) TrackAddToCart(session_id uint32, item_id uint, quantity uint) error {
	return ch.TrackEvent(session_id, PurchaseEvent, item_id, float32(quantity)*50.0)
}

func (ch *ClickHouse) TrackPurchase(session_id uint32, item_id uint, quantity uint) error {
	return ch.TrackEvent(session_id, PurchaseEvent, item_id, float32(quantity)*100.0)
}

func (ch *ClickHouse) TrackImpression(session_id uint32, item_id uint, position float32) error {
	return ch.TrackEvent(session_id, ImpressionEvent, item_id, position)
}

func (ch *ClickHouse) TrackEvent(session_id uint32, evt uint16, item_id uint, metric float32) error {
	ctx := context.Background()
	return ch.Conn.Exec(ctx, "INSERT INTO user_action (session_id, evt, item_id, metric, timestamp) VALUES (?, ?, ?, ?, ?)", session_id, evt, item_id, metric, time.Now())
}

//  timestamp DateTime,
// 		language String,
//     user_agent String,
//     ip String,

func (ch *ClickHouse) TrackSession(session_id uint32, r *http.Request) error {

	ctx := context.Background()
	return ch.Conn.Exec(ctx, "INSERT INTO user_session (session_id, timestamp, language, user_agent, ip) VALUES (?, ?, ?, ?, ?)", session_id, time.Now(), r.Header.Get("Accept-Language"), r.UserAgent(), r.RemoteAddr)
}
