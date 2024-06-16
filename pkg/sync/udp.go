package sync

import (
	"fmt"
	"log"
	"net"

	"tornberg.me/facet-search/pkg/index"
)

type UdpListener struct {
	Url  string
	conn *net.UDPConn
	addr *net.UDPAddr
}

func (s *UdpListener) Stop() error {
	return nil
}

func (s *UdpListener) Send(data []byte) error {
	if s.conn == nil {
		return fmt.Errorf("Connection not established")
	}
	_, err := s.conn.WriteToUDP([]byte("Hello UDP Client\n"), s.addr)
	return err
}

func (s *UdpListener) Start() error {
	udpAddr, err := net.ResolveUDPAddr("udp", s.Url)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	for {
		var buf [512]byte
		_, addr, err := conn.ReadFromUDP(buf[0:])
		s.addr = addr
		if err != nil {
			fmt.Println(err)
			return err
		}

		fmt.Print("> ", string(buf[0:]))

		// Write back the message over UPD

	}
}

type UdpSender struct {
	Url             string
	conn            *net.UDPConn
	DeleteChan      chan uint
	ItemChangedChan chan *index.DataItem
}

func (c *UdpSender) Connect() error {

	addr, err := net.ResolveUDPAddr("udp", c.Url)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	written, err := conn.Write([]byte("HELLO"[0:]))
	if err != nil {
		return err
	}
	log.Printf("Written %v bytes", written)
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

func (c *UdpSender) Send(data []byte) error {
	_, err := c.conn.Write(data)
	return err
}

func (c *UdpSender) Close() error {
	return c.conn.Close()
}

func NewUdpConnection(url string) *UdpSender {
	return &UdpSender{
		Url: url,
	}
}
