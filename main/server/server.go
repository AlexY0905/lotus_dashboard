package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
	client_ "yuncun.com/main/rpc"
	//client_ "yuncun.com/main/clientRPC"
)

type response struct {
	Result string `json:"result"`
	Error  *error `json:"error,omitempty"`
}

type webconn struct {
	conn     *websocket.Conn
	requests <-chan Lotus
	stop     <-chan struct{}
	writeLK  sync.Mutex
	exiting  chan struct{} //发送数据的通道
	//输入的信息
	incoming chan io.Reader
	err      error
}

func (c *webconn) handleConn() { //接收数据的通道
	c.incoming = make(chan io.Reader)
	go c.serverMessage()
	for {
		select {
		case r, ok := <-c.incoming:
			if !ok {
				if c.err != nil {
					if !websocket.IsCloseError(c.err, websocket.CloseNormalClosure) {
						client_.Log.Error("websocket error", c.err)
						//Error()
					}
					return
				}
			}
			var result response
			var re, _ = ioutil.ReadAll(r)
			if json.Unmarshal(re, &result) != nil {
				client_.Log.Error("handle me:", json.Unmarshal(re, &result))
				return
			} else {
				select {
				case <-c.exiting:
					client_.Log.Warn("failed to send request cancellation, websocket routing exited")
				}
			}
			go c.serverMessage()
		case req := <-c.requests:
			c.sendRequest(req)
		case <-c.stop:
			c.writeLK.Lock()
			cmsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			//写入关闭的错误信息
			if err := c.conn.WriteMessage(websocket.CloseMessage, cmsg); err != nil {
				client_.Log.Warn("failed to write close message:", err)
			}
			//关闭连接
			if err := c.conn.Close(); err != nil {
				client_.Log.Warnw("websocket close errpr", "error", err)
			}
			c.writeLK.Unlock()
			return
		}
	}

}
func (c *webconn) serverMessage() {
	msgType, r, err := c.conn.NextReader()
	if err != nil {
		c.err = err
		close(c.incoming)
		return
	}
	if msgType != websocket.BinaryMessage && msgType != websocket.TextMessage {
		c.err = errors.New("unsupported message type")
		close(c.incoming)
		return
	}
	c.incoming <- r
}

func (c *webconn) sendRequest(req Lotus) {
	c.writeLK.Lock()
	//建立连接
	if err := c.conn.WriteJSON(req); err != nil { //发送的是json数据
		client_.Log.Error("handle me", err)
		c.writeLK.Unlock()
	}
	c.writeLK.Unlock()
}

type Lotus struct {
	//id                        uint64       `json:"id"`
	LotusNetAddrsListenData   string       `json:"ip"`
	LotusIdData               string       `json:"nodeId,omitempty"`
	LotusChainHeadData        uint64       `json:"height,omitempty"`
	MinerData                 string       `json:"nodeName,omitempty"`
	LotusStateMinerPowerData  types.BigInt `json:"provenStorage,omitempty"`
	LotusWalletBalanceData    types.BigInt `json:"money,omitempty"`
	LotusStateMinerSectorSize uint64       `json:"sectorSize,omitempty"`
	CPU                       string       `json:"cpu,omitempty"`
	GPU                       string       `json:"gpu,omitempty"`
	Mem                       string       `json:"mem,omitempty"`
}

/*func (l *Lotus) NewTransmitClient(addr string, rerquestHeader http.Header) (client_.ClientCloser,error) {
	conn, _, err := websocket.DefaultDialer.Dial(addr, rerquestHeader)
	if err != nil {
		log.Fatal("dial:", err)
	}

	fmt.Println("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	stop := make(chan struct{})
	requests := make(chan Lotus, 100)
	exiting := make(chan struct{})
	select {
	  case requests <- *l:
		  fmt.Printf("send data")
	}
	c := webconn{
		conn: conn,
		requests: requests,
		stop: stop,
		exiting: exiting,
	}
	go c.handleConn()

	return func() {
		close(stop)
		<-c.exiting
	}, nil
}*/
func NewTransmitClient(addr string, rerquestHeader http.Header, l *Lotus) {
	//conn, _, err := websocket.DefaultDialer.Dial(addr, rerquestHeader)
	//conn, err := net.Dial("tcp", addr)
	client := &http.Client{Timeout: 20 * time.Second}
	/*if err != nil {
		fmt.Println("client dial err=", err)
		return
	}*/
	lotus, _ := json.Marshal(l)
	resp, err := client.Post(addr, "application/json", bytes.NewBuffer(lotus))
	//if i, err := conn.Write(lotus); err != nil {
	if err != nil {
		//	if err := c.conn.WriteMessage(websocket.TextMessage, jsondata); err != nil {
		fmt.Errorf("handle me", err)
	}
	result, _ := ioutil.ReadAll(resp.Body)
	var v interface{}
	json.Unmarshal(result, &v)
	fmt.Println(v)
}
