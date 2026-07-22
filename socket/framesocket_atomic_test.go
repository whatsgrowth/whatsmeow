package socket

import (
	"net/http"
	"sync"
	"testing"

	"github.com/coder/websocket"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func TestFrameSocketConnectionStateSupportsConcurrentReads(t *testing.T) {
	frameSocket := NewFrameSocket(waLog.Noop, http.DefaultClient)
	connection := &websocket.Conn{}
	var wait sync.WaitGroup

	for range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for range 1000 {
				_ = frameSocket.IsConnected()
			}
		}()
	}
	for range 1000 {
		frameSocket.conn.Store(connection)
		frameSocket.conn.Store(nil)
	}
	wait.Wait()
}
