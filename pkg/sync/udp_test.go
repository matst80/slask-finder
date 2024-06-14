package sync

import (
	"testing"
	"time"
)

func TestConnection(t *testing.T) {

	master := UdpSender{
		Url: ":8081",
	}
	client1 := UdpListener{
		Url: ":8081",
	}
	go func() {
		err := client1.Start()
		t.Logf("Server error: %s", err)
	}()
	time.Sleep(time.Millisecond * 30)
	go func() {
		master.Connect()
		bytes := make([]byte, 1024)
		l, err := master.conn.Read(bytes)
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
	client1.Send([]byte("Hello"))
	time.Sleep(time.Second * 5)
}
