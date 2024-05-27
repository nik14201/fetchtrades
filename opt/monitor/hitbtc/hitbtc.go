package hitbtc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gorilla/websocket"
)

// Request

// {
//     "method": "subscribe",
//     "ch": "trades",                         // Channel
//     "params": {
//         "symbols": ["ETHBTC", "BTCUSDT"],
//         "limit": 1
//     },
//     "id": 123
// }

// Response
// {
//     "result": {
//         "ch": "trades",                     // Channel
//         "subscriptions": ["ETHBTC", "BTCUSDT"]
//     },
//     "id": 123
// }

// Notification snapshot
// {
//     "ch": "trades",                         // Channel
//     "snapshot": {
//         "BTCUSDT": [{
//             "t": 1626861109494,             // Timestamp in milliseconds
//             "i": 1555634969,                // Trade identifier
//             "p": "30881.96",                // Price
//             "q": "12.66828",                // Quantity
//             "s": "buy"                      // Side
//         }]
//     }
// }

// Notification update
// {
//     "ch": "trades",
//     "update": {
//         "BTCUSDT": [{
//             "t": 1626861123552,
//             "i": 1555634969,
//             "p": "30877.68",
//             "q": "0.00006",
//             "s": "sell"
//         }]
//     }
// }

// type ResponseResult struct {
// 	CH            string
// 	Subscriptions []string
// }

// type Response struct {
// 	Result ResponseResult
// 	ID     uint64
// }

// // Notification snapshot
// type SnapshotData struct {
// 	Timestamp  uint64
// 	Identifier uint64
// 	Price      string
// 	Quantity   string
// 	Side       string
// }

// type SnapshotPair struct {
// 	Pair []SnapshotData
// }
// type SnapshotStruct struct {
// 	CH       string
// 	Snapshot SnapshotPair
// }

// // Notification update
// type UpdateData struct {
// 	Timestamp  uint64
// 	Identifier uint64
// 	Price      string
// 	Quantity   string
// 	Side       string
// }

// type UpdatePair struct {
// 	Pair []UpdateData
// }
// type UpdateStruct struct {
// 	CH     string
// 	Update UpdatePair
// }

// type MessageRead struct {
// 	Topic string `json:"topic"`
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

type List struct {
	Pair string
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
	msg := fmt.Sprintf(`
	{
		"method": "subscribe",
		"ch": "trades",   
		"params": {
			"symbols": %s,
			"limit": 1
		},
		"id": 123
	}
	
		`, list)
	//log.Println(msg)
	ws.Write(msg)
	go ws.ForRead()
}

func getListPair() string {
	resp, err := http.Get("https://api.hitbtc.com/api/2/public/orderbook")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var list_map map[string]interface{}
	var list string = "["
	json.Unmarshal(body, &list_map)
	var k int = 0
	for i, _ := range list_map {
		if k < len(list_map)-1 {
			list = list + fmt.Sprintf("\"%s\",", i)
		} else {
			list = list + fmt.Sprintf("\"%s\"", i)
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
	//u := url.URL{Scheme: "wss", Host: "api.hitbtc.com", Path: "/api/3/ws/public"}
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

func Iter(v interface{}) {
	iter := reflect.ValueOf(v).MapRange()
	for iter.Next() {
		keys := iter.Key()
		val := iter.Value()
		//log.Println(keys, reflect.Value(val).Interface().([]interface{})[0])
		iter_1 := reflect.ValueOf(reflect.Value(val).Interface().([]interface{})[0]).MapRange()
		for iter_1.Next() {
			key := iter_1.Key()
			val := iter_1.Value()
			//log.Println(key, val)
			if key.String() == "p" {
				pair := fmt.Sprintf("%v", keys)
				price := fmt.Sprintf("%v", val)
				log.Println("pair:", pair, "price:", price)
				mc.Set(&memcache.Item{Key: pair, Value: []byte(price)})
			}
		}
	}
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
		var mes_shanpshot map[string]interface{}
		if err := json.Unmarshal(messange, &mes_shanpshot); err != nil {
			panic(err)
		}
		if mes_shanpshot["snapshot"] != nil {
			Iter(mes_shanpshot["snapshot"])
		}
		if mes_shanpshot["update"] != nil {
			Iter(mes_shanpshot["update"])
		}
		time.Sleep(1 * time.Millisecond)
	}
}
