package bitfinex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gorilla/websocket"
)

type Response struct {
	Event   string
	Channel string
	ChanId  uint64
	Symbol  string
	Pair    string
}

type WebSock struct {
	Pair    string
	ws_conn *websocket.Conn
	Message Response
	mu_sock sync.Mutex
	buf     chan *websocket.Conn
}

var sock WebSock
var mu sync.Mutex
var AllTickerJson interface{}
var Conn int = 0

var memcached_host string = os.Getenv("MEMCACHED_HOST")
var mc *memcache.Client = memcache.New(memcached_host)

var mapPairChan map[string]string = make(map[string]string)

func StartWS() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Error bitfinex StartWS():", err)
			StartWS()
		}
	}()
	var ws WebSock
	if ws.ws_conn == nil {
		ws.Connect()
	}
	go ws.ForRead()
	ws.Write(`{"event": "subscribe",   "channel": "trades",   "pair": "BTCUSD" }`)

	list := getListPair()
	for _, l := range list {
		log.Println(l)
		if ws.ws_conn == nil {
			ws.Connect()
		}
		msg := fmt.Sprintf(`{"event": "subscribe",   "channel": "trades",   "pair": "%s" }`, l)
		log.Println(msg)
		ws.Write(msg)

	}

}

func getListPair() []string {
	resp, err := http.Get("https://api-pub.bitfinex.com/v2/conf/pub:list:pair:exchange")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var list [][]string
	json.Unmarshal(body, &list)
	return list[0]
}

func (ws *WebSock) Connect() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Error bitfinex Connect():", err)
		}
	}()
	Conn += 1
	log.Println("Conn = ", Conn)
	var err error
	env_scheme := os.Getenv("SCHEME")
	env_host := os.Getenv("HOST")
	env_path := os.Getenv("EXPATH")
	u := url.URL{Scheme: env_scheme, Host: env_host, Path: env_path}
	//u := url.URL{Scheme: "wss", Host: "api-pub.bitfinex.com", Path: "/ws/2"}
	if ws.ws_conn == nil {
		ws.mu_sock.Lock()
		ws.ws_conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		ws.mu_sock.Unlock()
	}
	if err != nil {
		log.Println("Error bitfinex Connect():", err)
	}
}

func (ws *WebSock) Write(msg string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Error bitfinex Write:", err)
			msg := fmt.Sprintf(`{"event": "subscribe",   "channel": "trades",   "pair": "%s" }`, ws.Pair)
			log.Println(msg)
			ws.Write(msg)
		}
	}()
	ws.mu_sock.Lock()
	log.Println(msg)
	err := ws.ws_conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		log.Println("Error bitfinex Write:", err)
	}
	ws.mu_sock.Unlock()
}

func (ws *WebSock) Read() []byte {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Error bitfinex Read():", err)
			ws.Read()
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

func getListTETUFloat64(v interface{}) float64 {
	z := reflect.ValueOf(v).Slice(3, 4).Index(0).Interface().(float64)
	return z
}

func getListFloat64(v interface{}) float64 {
	z := reflect.ValueOf(v).Slice(0, 1).Index(0).Interface().([]interface{})
	t := reflect.ValueOf(z[3]).Interface().(float64)
	return t
}

func getListChanId(v interface{}) string {
	s := fmt.Sprintf("%v", v)
	return s
}

func (ws *WebSock) ForRead() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Error bitfinex ForRead:", err)
			ws.ForRead()
		}
	}()
	for {
		if ws.ws_conn == nil {
			ws.Connect()
		}
		var response Response
		messange := ws.Read()
		if err := json.Unmarshal(messange, &response); err != nil {
			log.Println(err)
		} else {
			//fmt.Println("==== Response.ChanId ===", response.ChanId, response.Pair)
			mapPairChan[strconv.FormatUint(response.ChanId, 10)] = response.Pair
		}
		var list []interface{}
		if err := json.Unmarshal(messange, &list); err != nil {
			log.Println(err)
		}
		if list[1] == "te" || list[1] == "tu" {
			price_float64 := getListTETUFloat64(list[2])
			chan_id := getListChanId(list[0])
			pair := mapPairChan[chan_id]
			price := fmt.Sprintf("%v", price_float64)
			log.Println("===te tu== ", pair, "price: ", price)
			mc.Set(&memcache.Item{Key: pair, Value: []byte(price)})
		} else {
			if list[1] != "hb" {
				price_float64 := getListFloat64(list[1])
				chan_id := getListChanId(list[0])
				pair := mapPairChan[chan_id]
				price := fmt.Sprintf("%v", price_float64)
				log.Println(pair, "price: ", price)
				mc.Set(&memcache.Item{Key: pair, Value: []byte(price)})
			}
		}
		//log.Println("Time === mapPairChan", mapPairChan)
		time.Sleep(100 * time.Millisecond)
	}
}
