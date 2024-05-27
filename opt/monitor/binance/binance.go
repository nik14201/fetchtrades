package binance

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

type FetchTradesBinanceWS struct {
	Te string `json: "e"`
	TE uint64 `json: "E"`
	Ts string `json: "s"`
	Tt string `json: "t"`
	Tp string `json: "p"`
	Tq string `json: "q"`
	Tb uint64 `json: "b"`
	Ta uint64 `json: "a"`
	TT uint64 `json: "T"`
	Tm bool   `json: "m"`
	TM bool   `json: "M"`
}

type FetchTradesBinance struct {
	ID           uint64
	Price        decimal.Decimal
	Qty          decimal.Decimal
	QuoteQty     decimal.Decimal
	Time         string
	IsBuyerMaker bool
	IsBestMatch  bool
}

type WebSock struct {
	pair    string
	ws_conn *websocket.Conn
	//influxes            *InfluxdbTypes
	ExchangeInfoBinance interface{}
	mu_sock             sync.Mutex
	buf                 chan *websocket.Conn
	ListPair            []byte
}

var sock WebSock
var mu sync.Mutex
var Trades_binance map[string]FetchTradesBinance = make(map[string]FetchTradesBinance)
var AllTickerJson interface{}

var memcached_host string = os.Getenv("MEMCACHED_HOST")
var mc *memcache.Client = memcache.New(memcached_host)

func (ws *WebSock) StartWS() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_StartWS", "1 ccxt websocket", err)
			ws.StartWS()
		}
	}()
	ws.getListPair()
	ws.GetPairBinance()
}

func (ws *WebSock) getListPair() {
	resp, err := http.Get("https://api1.binance.com/api/v3/exchangeInfo")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	ws.ListPair = body
}

func (ws *WebSock) Connect() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_Connect", "2 ccxt websocket", err)
		}
	}()
	var err error
	env_scheme := os.Getenv("SCHEME")
	env_host := os.Getenv("HOST")
	stream_uuid := uuid.New()
	stream_path := fmt.Sprintf("/ws/stream_%s", stream_uuid)
	u := url.URL{Scheme: env_scheme, Host: env_host, Path: stream_path}
	//u := url.URL{Scheme: "wss", Host: "stream.binance.com:9443", Path: stream_path}
	if ws.ws_conn == nil {
		ws.mu_sock.Lock()
		ws.ws_conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		ws.mu_sock.Unlock()
	}
	if err != nil {
		log.Println("websocket_Connect", "4 ccxt websocket", err)
	}
}

func (ws *WebSock) WritePong() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_WritePong", "5 ccxt websocket", err)
			ws.Connect()
		}
	}()
	for {
		err := ws.ws_conn.WriteMessage(websocket.PongMessage, []byte{})
		if err != nil {
			log.Println("websocket_WritePong", "6 ccxt websocket", err)
			continue
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ws *WebSock) Write(s string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_Write", "allticker websocket", err)
		}
	}()
	msg := fmt.Sprintf(`{
	        "method":  "SUBSCRIBE",
	        "params":
	        [%s],
	        "id": 1
	        }`, s)
	ws.mu_sock.Lock()
	//log.Println(msg)
	err := ws.ws_conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		//log.Println("websocket_Write", "8 ccxt websocket", err)
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
	//log.Printf("recv: %s", message)
	return message
}

//=================================AllmarkPrice================================================

func (ws *WebSock) WriteAllmarkPrice() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_WriteAllmarkPrice", "allticker websocket", err)
		}
	}()
	ws.Write("!markPrice@arr")
}

func (ws *WebSock) ReadAllmarkPrice() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_WriteAllmarkPrice", "9911 ccxt websocket", err)
		}
	}()
	if ws.ws_conn == nil {
		return
	}
	var message []byte
	var err error
	ws.mu_sock.Lock()
	_, message, err = ws.ws_conn.ReadMessage()
	ws.mu_sock.Unlock()
	if len(message) > 0 {
		//log.Printf("recv: %s", message)
		err = json.Unmarshal([]byte(message), &AllTickerJson)
		if err != nil {
			log.Println("websocket_ReadAllmarkPrice", "11 ccxt websocket", err)
			return
		}
		return
		// mm := AllTickerJson.([]interface{})
		// for _, v := range mm {
		// 	m := v.(map[string]interface{})
		// 	//log.Println(m)
		// }
	}
}

//===============================AllTicker===============================

func (ws *WebSock) WriteAllTicker() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_WriteAllTicker", "allticker websocket", err)
		}
	}()
	ws.Write("!ticker@arr")
}

func (ws *WebSock) ReadAllTicker() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_ReadAllTicker", "99 ccxt websocket", err)
			os.Exit(1)
		}
	}()
	if ws.ws_conn == nil {
		return
	}
	var message []byte
	var err error
	//var trades FetchTradesBinance

	ws.mu_sock.Lock()
	_, message, err = ws.ws_conn.ReadMessage()
	ws.mu_sock.Unlock()
	if len(message) > 0 {
		//log.Printf("recv: %s", message)
		err = json.Unmarshal([]byte(message), &AllTickerJson)
		if err != nil {
			log.Println("websocket_ReadAllTicker", "11 ccxt websocket", err)
			return
		}
		// mm := AllTickerJson.([]interface{})
		// for _, v := range mm {
		// 	m := v.(map[string]interface{})
		// 	//log.Println(m)
		// }
	}
}

func (ws *WebSock) ReadMessageWS() error {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_ReadMessageWS", "9 ccxt websocket", err)
			log.Fatal(err)
			os.Exit(1)
		}
	}()
	//time.Sleep(200 * time.Millisecond)
	if ws.ws_conn == nil {
		ws.Connect()
		return nil
	}
	var message []byte
	var struct_json interface{}
	var trades FetchTradesBinance
	message = ws.Read()
	err := json.Unmarshal([]byte(message), &struct_json)
	if err != nil {
		mes := fmt.Sprintf("recv: %s", message)
		log.Println("websocket_ReadMessageWS_err", "9-err ccxt: ", mes)
		log.Println("websocket_ReadMessageWS_mes", "9-1 ccxt websocket", err)
		os.Exit(1)
	}
	m := struct_json.(map[string]interface{})
	pair := strings.ToLower(fmt.Sprintf("%v", m["s"]))
	price := fmt.Sprintf("%v", m["p"])
	qty := fmt.Sprintf("%v", m["q"])
	id := fmt.Sprintf("%v", m["t"])
	time := fmt.Sprintf("%v", m["T"])
	trades.Price, _ = decimal.NewFromString(price)
	trades.ID, _ = strconv.ParseUint(id, 10, 64)
	trades.Qty, _ = decimal.NewFromString(qty)
	trades.Time = time
	trades.IsBuyerMaker, _ = strconv.ParseBool(fmt.Sprintf("%v", m["m"]))
	trades.IsBestMatch, _ = strconv.ParseBool(fmt.Sprintf("%v", m["M"]))
	mu.Lock()
	Trades_binance[pair] = trades
	mu.Unlock()
	if price != "<nil>" {
		mc.Set(&memcache.Item{Key: pair, Value: []byte(price)})
		log.Println("========", pair, price)
	}
	return nil
}

func (ws *WebSock) GetPairBinance() {
	var pair_list string
	//data, err := ioutil.ReadFile("./binance/exchangeInfo")
	data := ws.ListPair
	json.Unmarshal(data, &ws.ExchangeInfoBinance)
	m := ws.ExchangeInfoBinance.(map[string]interface{})["symbols"].([]interface{})
	for _, v := range m {
		var w WebSock
		vv := v.(map[string]interface{})
		if strings.ToLower(fmt.Sprintf("%v", vv["symbol"])) == "btcusdt" {
			continue
		}
		pair := strings.ToLower(fmt.Sprintf("%v", vv["symbol"]))
		w.pair = pair
		pair = fmt.Sprintf("\"%s@aggTrade\" \n", pair)
		go w.Parse(pair)

	}
	pair_list = pair_list + "\"btcusdt@aggTrade\" "
	go ws.Parse(pair_list)
	pair_list = `
	"btcusdt@aggTrade",
	"btceth@aggTrade",
	"lstusdt@aggTrade",
	"ltcbtc@aggTrade",
	"ethusdt@aggTrade"  `
	go ws.Parse(pair_list)
	ws.pair = "btcusdt"
	go ws.Parse(pair_list)
}

func (ws *WebSock) Parse(list string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("websocket_ReadMessageWS", "9 ccxt websocket", err)
			log.Fatal(err)
			ws.Parse(list)
		}
	}()
	ws.Connect()
	ws.Write(list)
	for {
		switch ws.pair {
		case "btcusdt", "ethusdt", "ltcusdt", "ethbtc":
			time.Sleep(10 * time.Millisecond)
		default:
			time.Sleep(10 * time.Millisecond)
		}
		err := ws.ReadMessageWS()
		if err != nil {
			ws.ws_conn.Close()
			var w WebSock
			go w.Parse(list)
			break
		}
	}
}
