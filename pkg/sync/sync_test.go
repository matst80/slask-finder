package sync

import (
	"log"
	"net"
	"sync"
	"testing"

	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/search"
)

type BaseClient struct {
	Server *BaseServer
	Index  *index.Index
}

type BaseServer struct {
	Clients []*BaseClient
}

type UdpServer struct {
	Url         string
	remoteConns *sync.Map
	conn        *net.UDPConn
}

func (s *UdpServer) Stop() error {
	return nil
}

func (s *UdpServer) Send(data []byte) error {
	s.remoteConns.Range(func(key, value interface{}) bool {
		if _, err := s.conn.WriteTo(data, *value.(*net.Addr)); err != nil {
			s.remoteConns.Delete(key)
		}

		return true
	})
	return nil
}

func (s *UdpServer) Start() error {
	udpAddr, err := net.ResolveUDPAddr("udp", s.Url)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	s.remoteConns = new(sync.Map)

	for {
		buf := make([]byte, 1024)
		_, remoteAddr, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}

		if _, ok := s.remoteConns.Load(remoteAddr.String()); !ok {
			s.remoteConns.Store(remoteAddr.String(), &remoteAddr)
		}

		// Broadcast message to all connected clients
		go func() {
			s.remoteConns.Range(func(key, value interface{}) bool {
				if _, err := conn.WriteTo(buf, *value.(*net.Addr)); err != nil {
					s.remoteConns.Delete(key)

					return true
				}

				return true
			})
		}()
	}
}

type UdpClientToServerConnection struct {
	Url             string
	conn            *net.UDPConn
	DeleteChan      chan uint
	ItemChangedChan chan *index.DataItem
}

func (c *UdpClientToServerConnection) Connect() error {

	addr, err := net.ResolveUDPAddr("udp", c.Url)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	c.conn = conn
	go func() {
		bytes := make([]byte, 1024)
		for {
			len, err := c.conn.Read(bytes)
			if err != nil {
				log.Printf("Error reading from connection %v", err)
			}
			log.Printf("Received %v", string(bytes[:len]))
		}
	}()
	return nil
}

func (c *UdpClientToServerConnection) Send(data []byte) error {
	_, err := c.conn.Write(data)
	return err
}

func (c *UdpClientToServerConnection) Close() error {
	return c.conn.Close()
}

func NewUdpConnection(url string) *UdpClientToServerConnection {
	return &UdpClientToServerConnection{
		Url: url,
	}
}

func NewBaseServer() *BaseServer {
	return &BaseServer{
		Clients: []*BaseClient{},
	}
}

func (s *BaseServer) RegisterClient(client *BaseClient) {
	s.Clients = append(s.Clients, client)
}

func (s *BaseServer) ItemChanged(item *index.DataItem) {
	for _, client := range s.Clients {
		client.UpsertItem(item)
	}
}

func (s *BaseServer) ItemDeleted(item *index.DataItem) {
	for _, client := range s.Clients {
		client.DeleteItem(item.Id)
	}
}

func (s *BaseServer) ItemAdded(item *index.DataItem) {
	for _, client := range s.Clients {
		client.UpsertItem(item)
	}
}

func (c *BaseClient) UpsertItem(item *index.DataItem) {
	c.Index.UpsertItem(item)
}

func (c *BaseClient) DeleteItem(id uint) {
	c.Index.DeleteItem(id)
}

func TestSync(t *testing.T) {
	server := NewBaseServer()
	index1 := index.NewIndex(search.NewFreeTextIndex(&search.Tokenizer{MaxTokens: 128}))
	index2 := index.NewIndex(search.NewFreeTextIndex(&search.Tokenizer{MaxTokens: 128}))
	client1 := &BaseClient{Server: server, Index: index1}
	client2 := &BaseClient{Server: server, Index: index2}

	server.RegisterClient(client1)
	server.RegisterClient(client2)

	item := &index.DataItem{
		BaseItem: index.BaseItem{
			Id:    1,
			Title: "Test",
		},
		Fields: map[uint]string{
			1: "Test",
		},
	}

	server.ItemAdded(item)

	if _, ok := client1.Index.Items[1]; !ok {
		t.Error("Item not added to client 1")
	}

	if _, ok := client2.Index.Items[1]; !ok {
		t.Error("Item not added to client 2")
	}

	item.Fields[1] = "Test2"

	server.ItemChanged(item)

	if *client1.Index.Items[1].Fields[1].Value != "Test2" {
		t.Error("Item not updated on client 1")
	}

	if *client2.Index.Items[1].Fields[1].Value != "Test2" {
		t.Error("Item not updated on client 2")
	}

	server.ItemDeleted(item)

	if _, ok := client1.Index.Items[1]; ok {
		t.Error("Item not deleted from client 1")
	}

	if _, ok := client2.Index.Items[1]; ok {
		t.Error("Item not deleted from client 2")
	}
}
