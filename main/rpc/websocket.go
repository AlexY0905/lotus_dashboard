package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"reflect"
	"sync"
	"sync/atomic"
)

/*
---------------总方法，websocket的总方法
*/
/*
定义客户端连接的结构体
*/
type wsConn struct {
	conn     *websocket.Conn
	handler  handlers //handler中的rpcHandler 处理函数的结构体
	requests <-chan clientRequest
	stop     <-chan struct{}
	exiting  chan struct{} //发送数据的通道

	//输入的信息
	incoming    chan io.Reader
	incomingErr error

	//输出的信息
	writeLK sync.Mutex //共享变量 write的锁

	//我们发送到远程的请求的集合  int64表示的是请求的id
	inflight map[int64]clientRequest

	//chanHandlers是客户端通道处理程序的映射
	chanHandlers map[uint64]func(m []byte, ok bool)

	//handling 是我们需要处理的调用
	//CancelFunc告诉一个操作放弃它的工作。
	//CancelFunc不会等待工作停止。
	//在第一次调用之后，对CancelFunc的后续调用不执行任何操作。
	handling   map[int64]context.CancelFunc
	handlingLK sync.Mutex

	spawnOutChanHandlerOnce sync.Once

	//chanCtr是一个计数器，用于标识服务器端上的输出通道
	chanCtr uint64

	registerCh chan outChanReg //注册chan的通道

}

//输出通道的regular
type outChanReg struct {
	id uint64
	ch reflect.Value
}

//参数的结构struct
type frame struct {
	//common
	Jsonrpc string            `json:"jsonrpc"`
	ID      *int64            `json:"id,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
	//request
	Method string  `json:"method,omitempty"`
	Params []Param `json:"params,omitempty"` //[]param

	//response
	Result json.RawMessage `json:"result,omitempty"`
	Error  *respError      `json:"error,omitempty"`
}

//定义一些内建指令

const wsCancel = "xrpc.cancel"
const chValue = "xrpc.ch.val"
const chClose = "xrpc.ch.close"

/*
WebSocket的一些message的工具方法
*/

/*
//输出通道的regular
type outChanReg struct {
	id uint64
	ch reflect.Value
}
*/
func (c *wsConn) handleWsConn(ctx context.Context) {
	c.incoming = make(chan io.Reader)                     //接收数据的通道
	c.inflight = map[int64]clientRequest{}                //发送到远程的所有的请求
	c.handling = map[int64]context.CancelFunc{}           //需要处理的调用
	c.chanHandlers = map[uint64]func(m []byte, ok bool){} //chanHandlers是客户端通道处理程序的映射

	c.registerCh = make(chan outChanReg) //注册数据的通道

	//-----------------------------------------------------------------------------------------------------------defer的内容
	defer close(c.registerCh) //先关闭注册数据的通道
	defer close(c.exiting)    //关闭发送数据的通道
	/*
	   关闭通道需要确定得到了所有的调用的值，结束全文context
	*/
	defer func() {
		for id, req := range c.inflight { //req   request      //在handler类中 ready chan clientResponse 定义clientresponse的通道
			req.ready <- clientResponse{ //ready接收reaponse的通道
				Jsonrpc: "2.0",
				ID:      &id,
				Error: &respError{
					Message: "handler: websocket connection closed",
				},
			}

			c.handlingLK.Lock() //锁住共享内容
			for _, cancel := range c.handling {
				cancel()
			}
			c.handlingLK.Unlock() //释放共享内容
		}
	}()
	//---------------------------------------------------------------------------------------------------------------------------------
	//                                    开始进行连接

	go c.nextMessage() //c.incoming <- r

	//无限循环通道
	for {
		//go c.nextMessage()
		select {
		case r, ok := <-c.incoming: //从接收的通道取值赋值给r
			if !ok {
				if c.incomingErr != nil {
					if !websocket.IsCloseError(c.incomingErr, websocket.CloseNormalClosure) {
						Log.Error("websocket error", c.incomingErr)
					}
				}
				return //远程关闭
			}

			//处理所有的incoming 传入的消息(call 以及 response 调用和响应）的方式
			var frame frame

			// var re, _ = ioutil.ReadAll(r)
			//json.Unmarshal(re, &frame)
			//json.Unmarshal(re, &frame)
			//json.Unmarshal(m, &n)
			//fmt.Println(re)

			if err := json.NewDecoder(r).Decode(&frame); err != nil {
				Log.Error("handle me:", err)
				return
			}
			// var m json.RawMessage

			/*json.Unmarshal(frame.Result, &m)
			fmt.Println(frame.Result)
			fmt.Println(m)*/
			if len(frame.Params) == 2 {
				//	fmt.Println(frame.Params[0].Data)
				fmt.Println("asfassssssssssss")
			}
			c.handleFrame(ctx, frame)

			go c.nextMessage() // c.incoming <- r
		case req := <-c.requests: //发送数据的通道 是client中的clientrequest的通道struct返回handler中的request封装
			if &req.req.ID != nil { //返回clientRequest  对应的是client中的sendRequest通道中的方法
				c.inflight[*req.req.ID] = req //根据所有的请求拆分出一个个的request
			}
			c.sendRequest(req.req)
		case <-c.stop:
			c.writeLK.Lock()
			cmsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			//写入关闭的错误信息
			if err := c.conn.WriteMessage(websocket.CloseMessage, cmsg); err != nil {
				Log.Warn("failed to write close message:", err)
			}
			//关闭连接
			if err := c.conn.Close(); err != nil {
				Log.Warnw("websocket close errpr", "error", err)
			}
			c.writeLK.Unlock()
			return
		}
	}

	//-------------请求的内容

}

//nextMessage等待一条消息并将其放入传入接收exiting通道

func (c *wsConn) nextMessage() {
	msgType, r, err := c.conn.NextReader()
	if err != nil {
		c.incomingErr = err
		close(c.incoming)
		return
	}
	if msgType != websocket.BinaryMessage && msgType != websocket.TextMessage {
		c.incomingErr = errors.New("unsupported message type")
		close(c.incoming)
	}
	c.incoming <- r
}

//nextWriter等待writeLk并在获取锁时使用WS消息编写器调用cb回调 得到下一个写的编辑器
//回调函数
func (c *wsConn) nextWriter(cb func(writer io.Writer)) {
	c.writeLK.Lock()
	defer c.writeLK.Unlock()

	wcl, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		Log.Error("handle me:", err)
		return
	}
	cb(wcl)
	if err := wcl.Close(); err != nil {
		Log.Error("handle me:", err)
		return
	}
}

//向服务器发送请求的方法 sendRequest的方法

func (c *wsConn) sendRequest(req request) {
	//	fmt.Println(req)
	c.writeLK.Lock()
	//建立连接
	//jsondata, _ := json.Marshal(req)
	//c.conn.WriteMessage(websocket.TextMessage, jsondata)
	if err := c.conn.WriteJSON(&req); err != nil {
		//	if err := c.conn.WriteMessage(websocket.TextMessage, jsondata); err != nil {
		Log.Error("handle me", err)
		c.writeLK.Unlock()
	}
	c.writeLK.Unlock()
}

//处理response的函数
func (c *wsConn) handleResponse(frame frame) {
	if frame.ID == nil {
		return
	}
	req, ok := c.inflight[*frame.ID] //根据id得到请求
	if !ok {
		Log.Error("client got unknown ID in response")
		return
	}
	//retCh提供了用于处理传入通道消息的上下文和接收器
	//Result 提供了回复的内容
	if req.retCh != nil && frame.Result != nil {
		//output是通道   将result通过json的解析赋值给channel的id chid
		var channelid uint64
		if err := json.Unmarshal(frame.Result, &channelid); err != nil {
			Log.Errorf("failed to unmmarshal channel id response:%s,data '%s'", frame.Result)
			return
		}

		var chanCtx context.Context
		//retCh  是 client 中的makeChanSink 方法
		// chanHandlers 是一个map的函数  map[uint64]func(m []byte, ok bool)
		//type makeChanSink func() (context.Context, func([]byte, bool))  返回一个context类型还有一个func函数类型
		//将请求放入chanHandlers中
		chanCtx, c.chanHandlers[channelid] = req.retCh()
		go c.handleCtxAsync(chanCtx, *frame.ID) //请求websocketcancel的情况
	}

	//*******ready已经接收到了数据   ready
	req.ready <- clientResponse{
		Jsonrpc: frame.Jsonrpc,
		Result:  frame.Result,
		ID:      frame.ID,
		Error:   frame.Error,
	}
	delete(c.inflight, *frame.ID) //从发送到远程的请求的集合中删除已经得到请求信息的请求
}

//通过chanHandler处理的事件并且quit     （1）相关的channel关闭（2）也应该是一个goroutine（3）注意，只要我们正确地使用上下文(当不再使用异步函数时取消)，现在不这样做应该没有问题。
func (c *wsConn) handleCtxAsync(actx context.Context, id int64) {
	<-actx.Done()

	c.sendRequest(request{
		Jsonrpc: "2.0",
		Method:  wsCancel,
		Params:  []Param{{V: reflect.ValueOf(id)}},
		//Params: []reflect.Value{reflect.ValueOf(id)},
	})
}

// cancelCtx 是一个内建的rpc指令，它通过rpc处理取消上下文
func (c *wsConn) cancelCtx(req frame) {
	if req.ID != nil {
		Log.Warnf("%s call with id set，won't respond(有id集不会回复)", wsCancel)
	}
	var id int64 // yst correct
	if err := json.Unmarshal(req.Params[0].Data, &id); err != nil {
		Log.Error("handle me:", err)
		return
	}
	c.handlingLK.Lock()

	defer c.handlingLK.Unlock()
	cf, ok := c.handling[id] //	handling   map[int64]context.CancelFunc
	if ok {
		cf()
	}
}

//handleOutChans处理服务器端上的通道通信
////(将通道消息转发给客户端)
func (c *wsConn) handleOutChans() {
	regV := reflect.ValueOf(c.registerCh) //registerCh chan outChanReg //注册chan的通道

	cases := []reflect.SelectCase{
		{ // registration chan always 0
			Dir:  reflect.SelectRecv,
			Chan: regV,
		},
	}
	var caseToID []uint64

	//循环接收服务器的信息
	for {
		chosen, val, ok := reflect.Select(cases)

		if chosen == 0 { // control channel
			if !ok {
				// control channel closed - signals closed connection
				//
				// We're not closing any channels as we're on receiving end.
				// Also, context cancellation below should take care of any running
				// requests
				return
			}

			registration := val.Interface().(outChanReg)

			caseToID = append(caseToID, registration.id)
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: registration.ch,
			})
			continue
		}

		if !ok {
			// Output channel closed, cleanup, and tell remote that this happened

			n := len(caseToID)
			if n > 0 {
				cases[chosen] = cases[n]
				caseToID[chosen-1] = caseToID[n-1]
			}

			id := caseToID[chosen-1]
			cases = cases[:n]
			caseToID = caseToID[:n-1]

			c.sendRequest(request{ //发送请求
				Jsonrpc: "2.0",
				ID:      nil, // notification
				Method:  chClose,
				Params:  []Param{{V: reflect.ValueOf(id)}},
				//Params: []reflect.Value{reflect.ValueOf(id)},
			})
			continue
		}

		// forward message
		c.sendRequest(request{
			Jsonrpc: "2.0",
			ID:      nil, // notification
			Method:  chValue,
			Params:  []Param{{V: reflect.ValueOf(caseToID[chosen-1])}, {V: val}},
			//Params: []reflect.Value{reflect.ValueOf(caseToID[chosen-1]), val},
		})
	}
}

//handleChanOut注册输出通道以转发给客户端
func (c *wsConn) handleChanOut(ch reflect.Value) interface{} {
	//sync.Once
	c.spawnOutChanHandlerOnce.Do(func() {
		go c.handleOutChans()
	})
	id := atomic.AddUint64(&c.chanCtr, 1)

	//注册channel的通道
	c.registerCh <- outChanReg{
		id: id,
		ch: ch,
	}
	return id
}

//main handling logic  处理保存在 c.chanHandlers[chid]的channel
func (c *wsConn) handleChanMessage(frame frame) {
	var chid uint64 //  yst correct
	if err := json.Unmarshal(frame.Params[0].Data, &chid); err != nil {
		Log.Error("failed to unmarshal channel id in xrpc.ch.val:s%", err)
	}
	hnd, ok := c.chanHandlers[chid] //从chanhandler中找
	if !ok {
		Log.Errorf("xrpc.ch.val:handler %d not found", chid)
		return
	}
	hnd(frame.Params[1].Data, true)
}

//channel  关闭的方法
func (c *wsConn) handleChanClose(frame frame) {
	var chid uint64
	if err := json.Unmarshal(frame.Params[0].Data, &chid); err != nil {
		Log.Error("failed to unmarshal channel id in xrpc.ch.val :%s", err)
		return
	}
	hnd, ok := c.chanHandlers[chid]
	if !ok {
		Log.Errorf("xrpc.ch.val: handler %d not found", chid)
		return
	}

	delete(c.chanHandlers, chid) //已经处理完从chanHandler中删除

	hnd(nil, false) //删除之后变为nil
}

//handleCall方法处理调用的方法
func (c *wsConn) handleCall(ctx context.Context, frame frame) {
	req := request{
		Jsonrpc: frame.Jsonrpc,
		ID:      frame.ID,
		Meta:    frame.Meta,
		Method:  frame.Method,
		Params:  frame.Params,
	}

	ctx, cancel := context.WithCancel(ctx) //从上下文中获取是否cancel
	// 初始化一个nextWriter方法对象
	nextWriter := func(cb func(writer io.Writer)) {
		cb(ioutil.Discard) //回调函数ioutil.Discard是一个io.Writer变量，对它的所有写调用都成功，不做任何操作。
	}
	//初始化一个done的方法对象
	done := func(keepCtx bool) {
		if !keepCtx {
			cancel()
		}
	}
	if &frame.ID != nil {
		nextWriter = c.nextWriter
		c.handlingLK.Lock()
		c.handling[*frame.ID] = cancel
		c.handlingLK.Unlock()

		done = func(keepCtx bool) {
			c.handlingLK.Lock()
			defer c.handlingLK.Unlock()

			if !keepCtx {
				cancel()
				delete(c.handling, *frame.ID)
			}
		}
	}
	go c.handler.handle(ctx, req, nextWriter, rpcError, done, c.handleChanOut)
}

//处理所有的incoming 传入的消息(call 以及 response 调用和响应）

func (c *wsConn) handleFrame(ctx context.Context, frame frame) {
	//得到message 的类型从方法的名字
	//"" 表示的是response
	//xrpc.* 表示的是内建指令，指的是内建的指令的执行，自己定义的指令
	//其余的表示的是incoming传入的远程的调用

	switch frame.Method {
	case "":
		c.handleResponse(frame)
	case wsCancel:
		c.cancelCtx(frame)
	case chValue:
		c.handleChanMessage(frame)
	case chClose:
		c.handleChanClose(frame)
	default:
		c.handleCall(ctx, frame) //服务器端默认走的这个方法
	}
}
