package kraken

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
		"event": "subscribe",
		"pair": %s,
		"subscription": {
		  "name": "trade"
		}
	  }
		`, list)
	log.Println(msg)
	ws.Write(msg)
	go ws.ForRead()
}

type ListDat struct {
	Altname string
	Wsname  string
}

type ListRes struct {
	Error  string
	Result map[string]ListDat
}

func getListPair() string {
	resp, err := http.Get("https://api.kraken.com/0/public/AssetPairs")
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
	for i, _ := range list_map.Result {
		if k < len(list_map.Result)-1 {
			list = list + fmt.Sprintf("\"%s\",", list_map.Result[i].Wsname)
			log.Println(list)
		} else {
			list = list + fmt.Sprintf("\"%s\"", list_map.Result[i].Wsname)
			log.Println(list)
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
	//u := url.URL{Scheme: "wss", Host: "ws.kraken.com", Path: "/api/3/ws/public"}
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
		var mes []interface{}
		if err := json.Unmarshal(messange, &mes); err != nil {
			panic(err)
		}
		log.Println(mes[3])
		pair := fmt.Sprintf("%v", mes[3])
		ref_1 := reflect.ValueOf(mes[1]).Index(0).Interface().([]interface{})
		ref_2 := reflect.ValueOf(ref_1).Slice(0, 1).Index(0)
		price := fmt.Sprintf("%v", ref_2)
		log.Println(pair, price)
		mc.Set(&memcache.Item{Key: pair, Value: []byte(price)})
		time.Sleep(1 * time.Millisecond)
	}
}
