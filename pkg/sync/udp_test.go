package sync

import (
	"testing"
	"time"
)

func TestConnection(t *testing.T) {

	client := UdpClientToServerConnection{
		Url: ":8081",
	}
	server := UdpServer{
		Url: ":8081",
	}
	go func() {
		err := server.Start()
		t.Logf("Server error: %s", err)
	}()
	time.Sleep(time.Millisecond * 30)
	go func() {
		client.Connect()
		bytes := make([]byte, 1024)
		l, err := client.conn.Read(bytes)
		if l == 0 {
			t.Errorf("Error reading from connection: %s", err)
		}
		if err != nil {
			t.Errorf("Error reading from connection: %s", err)
		}
		if string(bytes) != "Hello" {
			t.Errorf("Expected: Hello, got: %s", string(bytes))
		}
	}()

	time.Sleep(time.Second)
	server.Send([]byte("Hello"))
	time.Sleep(time.Second * 5)
}
