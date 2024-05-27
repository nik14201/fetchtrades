package bybit

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gorilla/websocket"
)

type MesData struct {
	Trade_time_ms  uint64  `json:"trade_time_ms"`
	Timestamp      string  `json:"timestamp"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	Size           int     `json:"size"`
	Price          float64 `json:"price"`
	Tick_direction string  `json:"tick_direction"`
	Trade_id       string  `json:"trade_id"`
	Cross_seq      uint64  `json:"cross_seq"`
}

type MessageRead struct {
	Topic string    `json:"topic"`
	Data  []MesData `json:"data"`
}

type WebSock struct {
	pair                string
	ws_conn             *websocket.Conn
	ExchangeInfoBinance interface{}
	mu_sock             sync.Mutex
	buf                 chan *websocket.Conn
}

var sock WebSock
var mu sync.Mutex
var AllTickerJson interface{}
var Conn int = 0

var memcached_host string = os.Getenv("MEMCACHED_HOST")
var mc *memcache.Client = memcache.New(memcached_host)

func (ws *WebSock) StartWS() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_StartWS", "1 ccxt websocket", err)
			ws.StartWS()
		}
	}()
	if ws.ws_conn == nil {
		ws.Connect()
	}
	go ws.ForRead()
	//ws.Write(`{"op": "subscribe", "args": ["trade.BTCUSD|XRPUSD|ETHUSD|ETHBTC"]}`)
	ws.Write(`{"op": "subscribe", "args": ["trade"]}`)

}

func (ws *WebSock) Connect() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_Connect", "2 ccxt websocket", err)
		}
	}()
	Conn += 1
	log.Println("Conn = ", Conn)
	var err error
	env_scheme := os.Getenv("SCHEME")
	env_host := os.Getenv("HOST")
	env_path := os.Getenv("EXPATH")
	u := url.URL{Scheme: env_scheme, Host: env_host, Path: env_path}
	//u := url.URL{Scheme: "wss", Host: "stream.bybit.com", Path: "/realtime"}
	if ws.ws_conn == nil {
		ws.mu_sock.Lock()
		ws.ws_conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		ws.mu_sock.Unlock()
	}
	if err != nil {
		log.Println("websocket_Connect", "4 ccxt websocket", err)
	}
}

func (ws *WebSock) Write(msg string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Error write websocket bybit: ", err)
		}
	}()
	ws.mu_sock.Lock()
	log.Println(msg)
	err := ws.ws_conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		log.Println("Error write websocket bybit: ", err)
	}
	ws.mu_sock.Unlock()
}

func (ws *WebSock) Read() []byte {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_Read", "99 ccxt websocket", err)
		}
	}()
	if ws.ws_conn == nil {
		ws.Connect()
	}
	var message []byte
	ws.mu_sock.Lock()
	_, message, _ = ws.ws_conn.ReadMessage()
	ws.mu_sock.Unlock()
	return message
}

func (ws *WebSock) ForRead() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_Read", "99 ccxt websocket", err)
		}
		ws.ForRead()
	}()
	for {
		messange := ws.Read()
		var mes MessageRead
		if err := json.Unmarshal(messange, &mes); err != nil {
			panic(err)
		}
		pair := mes.Data[0].Symbol
		price := fmt.Sprintf("%f", mes.Data[0].Price)
		if price != "<nil>" {
			mc.Set(&memcache.Item{Key: pair, Value: []byte(price)})
			log.Println("========", pair, price)
		}
		time.Sleep(1 * time.Millisecond)
	}
}
