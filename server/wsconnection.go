package server

import (
	"context"
	"net"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/rohit-sonawane/opamp-go/internal"
	"github.com/rohit-sonawane/opamp-go/protobufs"
	"github.com/rohit-sonawane/opamp-go/server/types"
)

// wsConnection represents a persistent OpAMP connection over a WebSocket.
type wsConnection struct {
	// The websocket library does not allow multiple concurrent write operations,
	// so ensure that we only have a single operation in progress at a time.
	// For more: https://pkg.go.dev/github.com/gorilla/websocket#hdr-Concurrency
	connMutex *sync.Mutex
	wsConn    *websocket.Conn
}

var _ types.Connection = (*wsConnection)(nil)

func (c wsConnection) Connection() net.Conn {
	return c.wsConn.UnderlyingConn()
}

func (c wsConnection) Send(_ context.Context, message *protobufs.ServerToAgent) error {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()

	return internal.WriteWSMessage(c.wsConn, message)
}

func (c wsConnection) Disconnect() error {
	return c.wsConn.Close()
}
