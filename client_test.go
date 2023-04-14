package asendexClient

import (
	"reflect"
	"testing"

	"github.com/GritselMaks/bk_test/mocks"
)

func Test_getChannelName(t *testing.T) {
	tests := []struct {
		name   string
		symbol string
		want   string
	}{
		{
			name:   "valid",
			symbol: "USDT_BTC",
			want:   "bbo:USDT/BTC",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getChannelName(tt.symbol); got != tt.want {
				t.Errorf("getChannelName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_prepareOrder(t *testing.T) {
	tests := []struct {
		name    string
		s       []string
		want    Order
		wantErr bool
	}{
		{
			name: "case_valid",
			s: []string{"9309.11",
				"0.0197172"},
			want:    Order{Price: 9309.11, Amount: 0.0197172},
			wantErr: false,
		},
		{
			name: "case_valid_2",
			s: []string{"9309",
				"197172"},
			want:    Order{Price: 9309, Amount: 197172},
			wantErr: false,
		},
		{
			name: "case_invalid",
			s: []string{"sd",
				"dd"},
			want:    Order{},
			wantErr: true,
		},
		{
			name:    "case_invalid_2",
			s:       []string{"sd"},
			want:    Order{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := prepareOrder(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prepareOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Connection(t *testing.T) {
	mockConn := mocks.NewConnection(t)

	//not error case
	mockConn.On("WSConnect", "url").Once().Return(nil)
	mockConn.On("Read").Once().Return(nil, nil)
	testClient := &Client{
		done: make(chan struct{}),
		conn: mockConn,
		url:  "url",
	}
	if err := testClient.Connection(); err != nil {
		t.Errorf("Client.Connection() error = %v", err)
	}

	// error case
	mockConn.On("WSConnect", "").Return(ErrConnectionServer)
	testClient.url = ""
	if err := testClient.Connection(); err == nil {
		t.Errorf("Client.Connection() error = %v", err)
	}

	// error case read message
	mockConn.On("WSConnect", "url").Return(nil)
	mockConn.On("Read").Return(nil, ErrConnectionClose)
	testClient.url = "url"
	if err := testClient.Connection(); err == nil {
		t.Errorf("Client.Connection() error = %v", err)
	}
}

func TestClient_Disconnect(t *testing.T) {
	mockConn := mocks.NewConnection(t)
	testClient := &Client{
		done: make(chan struct{}),
		conn: mockConn,
		url:  "url",
	}
	mockConn.On("Close").Return(nil)
	testClient.Disconnect()
}

func TestClient_SubscribeToChannel(t *testing.T) {
	message := `{ "op": "sub", "ch":"bbo:symbol" }`
	//error write message
	mockConn := mocks.NewConnection(t)
	testClient := &Client{
		done: make(chan struct{}),
		conn: mockConn,
		url:  "url",
	}
	//write error connection
	mockConn.On("Write", 1, []byte(message)).Once().Return(ErrConnectionClose)
	mockConn.On("Close").Once().Return(nil)
	if err := testClient.SubscribeToChannel("symbol"); err == nil {
		t.Errorf("Client.SubscribeToChannel() error = %v", err)
	}

	//error read message
	testClient = &Client{
		done: make(chan struct{}),
		conn: mockConn,
		url:  "url",
	}
	mockConn.On("Write", 1, []byte(message)).Once().Return(nil)
	mockConn.On("Close").Once().Return(nil)
	mockConn.On("Read").Once().Return(nil, ErrConnectionClose)
	if err := testClient.SubscribeToChannel("symbol"); err == nil {
		t.Errorf("Client.SubscribeToChannel() error = %v", err)
	}

	//error message
	testClient = &Client{
		done: make(chan struct{}),
		conn: mockConn,
		url:  "url",
	}
	responceMessage := `{ "op": "sub", "ch":"bbo:symbol", "code":10000 }`
	mockConn.On("Write", 1, []byte(message)).Once().Return(nil)
	mockConn.On("Close").Once().Return(nil)
	mockConn.On("Read").Once().Return([]byte(responceMessage), nil)
	if err := testClient.SubscribeToChannel("symbol"); err == nil {
		t.Errorf("Client.SubscribeToChannel() error = %v", err)
	}

	//Success
	testClient = &Client{
		done: make(chan struct{}),
		conn: mockConn,
		url:  "url",
	}
	responceMessage = `{ "op": "sub", "ch":"bbo:symbol", "code":0 }`
	mockConn.On("Write", 1, []byte(message)).Return(nil)
	mockConn.On("Read").Return([]byte(responceMessage), nil)
	if err := testClient.SubscribeToChannel("symbol"); err != nil {
		t.Errorf("Client.SubscribeToChannel() error = %v", err)
	}

}

func TestClient_connectionAliveMessage(t *testing.T) {
	message := `{ "op": "ping" }`
	//error write message
	mockConn := mocks.NewConnection(t)
	testClient := &Client{
		done: make(chan struct{}),
		conn: mockConn,
		url:  "url",
	}
	mockConn.On("Write", 1, []byte(message)).Return(nil)
	testClient.connectionAliveMessage("ping")
}

func TestClient_ReadMessagesFromChannel(t *testing.T) {
	testCh := make(chan BestOrderBook)
	badFormatMessage := ""
	correctMessage := `{
		"m": "bbo",
		"symbol": "BTC/USDT",
		"data": {
			"ts": 1573068442532,
			"bid": [
				"9309.11",
				"0.0197172"
			],
			"ask": [
				"9309.12",
				"0.8851266"
			]
		}
	}`
	pingMessage := `{ "m": "ping", "hp": 3 }`
	mockConn := mocks.NewConnection(t)
	testClient := &Client{
		done: make(chan struct{}),
		conn: mockConn,
		url:  "url",
	}
	//read bad format message
	mockConn.On("Read").Once().Return([]byte(badFormatMessage), nil)
	testClient.ReadMessagesFromChannel(testCh)
	//read ping message
	mockConn.On("Read").Once().Return([]byte(pingMessage), nil)
	mockConn.On("Write", 1, []byte(`{ "op": "pong" }`)).Once().Return(nil)
	mockConn.On("Read").Once().Return(nil, nil)
	testClient.ReadMessagesFromChannel(testCh)

	//read correct message
	mockConn.On("Read").Once().Return([]byte(correctMessage), nil)
	mockConn.On("Read").Once().Return(nil, nil)
	testClient.ReadMessagesFromChannel(testCh)

	//error read from channel.
	mockConn.On("Read").Once().Return(nil, ErrConnectionClose)
	testClient.ReadMessagesFromChannel(testCh)
}
