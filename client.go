package asendexClient

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	url  string
	done chan struct{}
	conn Connection
}

// NewClient creates a new Client.
func NewClient(url string) *Client {
	return &Client{url: url,
		done: make(chan struct{}),
		conn: NewConnection()}
}

// Connection is connect to WebSocket server
func (c *Client) Connection() error {
	if err := c.conn.WSConnect(c.url); err != nil {
		return err
	}
	// read confirmation message
	_, err := c.conn.Read()
	if err != nil {
		return err
	}
	return nil
}

// Disconnect from WebSocket
func (c *Client) Disconnect() {
	close(c.done)
	<-time.After(time.Second)
	c.conn.Close()
}

// Function SubscribeToChannel subscribe to updates of BBO for a given symbol.
// The symbol must be of the form "TOKEN_ASSET"
// As an example "USDT_BTC" where USDT is TOKEN and BTC is ASSET
// If subscription not succesfull, return error and disconect from WebSocket
func (c *Client) SubscribeToChannel(symbol string) error {
	subscription := fmt.Sprintf(`{ "op": "sub", "ch":"%s" }`, getChannelName(symbol))
	if err := c.conn.Write(1, []byte(subscription)); err != nil {
		c.Disconnect()
		return err
	}
	// read confirmation message. If error Code not equal 0, return error and close connection
	msg, err := c.conn.Read()
	if err != nil {
		c.Disconnect()
		return err
	}
	var message struct {
		M      string `json:"m"`
		Ch     string `json:"ch"`
		Code   int    `json:"code"`
		Reason string `json:"reason,omitempty"`
	}
	err = json.Unmarshal(msg, &message)
	if err != nil || message.Code != 0 {
		c.Disconnect()
		return fmt.Errorf("subscription error: %v, %s", message.Code, message.Reason)
	}
	return nil
}

// Function ReadMessagesFromChannel read data from websocket and write that
// we receive to the channel.
func (c *Client) ReadMessagesFromChannel(ch chan<- BestOrderBook) {
	for {
		msg, err := c.conn.Read()
		if err != nil {
			switch err {
			case ErrConnectionClose:
				log.Printf("error Read message from server. Conncetion: %s", err)
				return
			default:
				log.Printf("error Read message from server: %s", err)
				c.Disconnect()
				return
			}
		}
		var message struct {
			M      string `json:"m"`
			Symbol string `json:"symbol"`
			Data   struct {
				Ts  int64    `json:"ts"`
				Bid []string `json:"bid"`
				Ask []string `json:"ask"`
			} `json:"data"`
		}

		err = json.Unmarshal(msg, &message)
		if err != nil {
			log.Printf("error parsing message: %s\n", err)
			return
		}
		switch message.M {
		case "ping":
			//send "pong" message for keeping alive connection
			c.connectionAliveMessage("pong")
		case "bbo":
			orderBook, err := prepareBestBook(message.Data.Bid, message.Data.Ask)
			if err != nil {
				log.Printf("error prepareBestBook: %s\n", err)
				break
			}
			go func() {
				select {
				case <-c.done:
					return
				case ch <- orderBook:
				}
			}()
		default:
		}
	}

}

// WriteMessagesToChannel support alive websocket connection.
// Client initiated ping message.
// It's unblocked function.
func (c *Client) WriteMessagesToChannel() {
	c.connectionAliveMessage("ping")
}

// connectionAliveMessage send connection Allive Message
func (c *Client) connectionAliveMessage(msg string) {
	ping := `{ "op": "%s" }`
	if err := c.conn.Write(1, []byte(fmt.Sprintf(ping, msg))); err != nil {
		log.Println("error sending Connection Keep Alive message")
	}
}

// getChannelName conver symbol to channel name
func getChannelName(symbol string) string {
	return fmt.Sprintf("bbo:%s", strings.Replace(symbol, "_", "/", 1))
}

// prepareOrder create new Order from slice of strings. Slice must have two elements.
// First value is Price and second is Amount.
func prepareOrder(s []string) (Order, error) {
	var order Order
	if len(s) != 2 {
		return order, fmt.Errorf("invalid input value")
	}
	for i, str := range s {
		value, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return order, fmt.Errorf("invalid input value: %w", err)
		}
		if i == 0 {
			order.Price = value
		} else {
			order.Amount = value
		}
	}
	return order, nil
}

// prepareBestBook create new BestOrderBook. It can return error.
func prepareBestBook(bidStr, askStr []string) (BestOrderBook, error) {
	bid, err := prepareOrder(bidStr)
	if err != nil {
		return BestOrderBook{}, err
	}
	ask, err := prepareOrder(askStr)
	if err != nil {
		return BestOrderBook{}, err
	}
	return BestOrderBook{
		Ask: ask,
		Bid: bid,
	}, nil
}
