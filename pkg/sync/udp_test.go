package sync

import (
	"testing"
)

func TestConnection(t *testing.T) {

	client := UdpClientToServerConnection{
		Url: "localhost:8081",
	}
	server := UdpServer{
		Url: "localhost:8081",
	}
	go server.Start()
	client.Connect()
	server.Send([]byte("Hello"))
}
