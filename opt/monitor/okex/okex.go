package okex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gorilla/websocket"
)

// Request Example

// {
//   "op": "subscribe",
//   "args": [
//     {
//       "channel": "trades",
//       "instId": "BTC-USD-191227"
//     }
//   ]
// }

// Request parameters
// Parameter 	Type 	Required 	Description
// op 	String 	Yes 	Operation, subscribe unsubscribe
// args 	Array 	Yes 	List of subscribed channels
// > channel 	String 	Yes 	Channel name, trades
// > instId 	String 	Yes 	Instrument ID

//     Successful Response Example

// {
//   "event": "subscribe",
//   "args": {
//       "channel": "trades",
//       "instId": "BTC-USD-191227"
//     }
// }

//     Failure Response Example

// {
//   "event": "error",
//   "code": "60012",
//   "msg": "Unrecognized request: {\"op\": \"subscribe\", \"argss\":[{ \"channel\" : \"trades\", \"instId\" : \"BTC-USD-191227\"}]}"
// }

// Response parameters
// Parameter 	Type 	Required 	Description
// event 	String 	Yes 	Event, subscribe unsubscribe error
// arg 	Object 	No 	Subscribed channel
// > channel 	String 	Yes 	Channel name
// > instId 	String 	Yes 	Instrument ID
// code 	String 	No 	Error code
// msg 	String 	No 	Error message

//     Push Data Example

// {
//   "arg": {
//     "channel": "trades",
//     "instId": "BTC-USDT"
//   },
//   "data": [
//     {
//       "instId": "BTC-USDT",
//       "tradeId": "130639474",
//       "px": "42219.9",
//       "sz": "0.12060306",
//       "side": "buy",
//       "ts": "1630048897897"
//     }
//   ]
// }

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

type RespDat struct {
	InstId  string
	TradeId string
	PX      string
	SZ      string
	Side    string
	Ts      string
}

type Response struct {
	Arg  interface{}
	Data []RespDat
}

type ListDat struct {
	InstId string
}

type ListRes struct {
	Code string
	Msg  string
	Data []ListDat
}

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

	list := getListPair()
	fmt.Println(list)
	msg := fmt.Sprintf(`
	{
		"op": "subscribe",
		"args": %s
	}
	`, list)
	log.Println(msg)
	ws.Write(msg)
	go ws.ForRead()
}

func getListPair() string {
	resp, err := http.Get("https://www.okx.com/api/v5/market/tickers?instType=SPOT")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var list_map ListRes
	var list string = "["
	json.Unmarshal(body, &list_map)
	var k int = 0
	for i, _ := range list_map.Data {
		if k < len(list_map.Data)-1 {
			list = list + fmt.Sprintf(`
			{
				"channel": "trades",
				"instId": "%s"
			},`, list_map.Data[i].InstId)
		} else {
			list = list + fmt.Sprintf(`
			{
			   "channel": "trades",
			   "instId": "%s"
			}`, list_map.Data[i].InstId)
		}
		k += 1
	}
	list = list + "]"
	return list
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
		if ws.ws_conn == nil {
			ws.Connect()
		}
		messange := ws.Read()
		var mes Response
		if err := json.Unmarshal(messange, &mes); err != nil {
			panic(err)
		}
		pair := fmt.Sprintf("%v", mes.Data[0].InstId)
		price := fmt.Sprintf("%v", mes.Data[0].PX)
		log.Println(pair, price)
		mc.Set(&memcache.Item{Key: pair, Value: []byte(price)})
		time.Sleep(1 * time.Millisecond)
	}
}
