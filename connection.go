package asendexClient

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var ErrConnectionClose = errors.New("client is not connected")
var ErrConnectionServer = errors.New("error connecting to server")

//go:generate mockery --name Connection
type Connection interface {
	WSConnect(url string) error
	Read() ([]byte, error)
	Write(messageType int, data []byte) error
	Close() error
}

type connectionImpl struct {
	conn *websocket.Conn
}

func NewConnection() Connection {
	return &connectionImpl{}
}

func (c *connectionImpl) WSConnect(url string) error {
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{})
	if err != nil {
		return fmt.Errorf("%s: %w", ErrConnectionServer.Error(), err)
	}
	c.conn = conn
	return nil
}

func (c *connectionImpl) Read() ([]byte, error) {
	if c.conn == nil {
		return nil, ErrConnectionClose
	}
	_, message, err := c.conn.ReadMessage()
	return message, err
}

func (c *connectionImpl) Write(messageType int, data []byte) error {
	if c.conn == nil {
		return ErrConnectionClose
	}
	return c.conn.WriteMessage(messageType, data)
}

func (c *connectionImpl) Close() error {
	if c.conn == nil {
		return ErrConnectionClose
	}
	return c.conn.Close()
}
