package rpc

import (
	"container/list"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/lotus/chain/address"
	type_ "github.com/filecoin-project/lotus/chain/types"
	"github.com/gorilla/websocket"
	logging "github.com/ipfs/go-log"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
	"log"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
)

var Log = logging.Logger("rpc")

/*
 客户端回复的结构体
*/
type clientResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	ID      *int64          `json:"id"` //回复的id
	Error   *respError      `json:"error,omitempty"`
}

type MinerPower struct {
	MinerPower type_.BigInt
	TotalPower type_.BigInt
}

type MinerSectors struct {
	Pset uint64
	Sset uint64
}

/*
客户端请求的数据的结构体
*/
type clientRequest struct {
	req   request             //在handler类中
	ready chan clientResponse //定义clientresponse的接收通道
	retCh makeChanSink        //retCh提供了用于处理传入通道消息的上下文和接收器
}

/*
client 客户端的结构体
*/
type client struct {
	namespace string
	requests  chan clientRequest //定义了关于客户端请求的一个通道
	exiting   <-chan struct{}    //定义了关于接收数据的通道
	idCtr     int64              //默认为3
}

type ErrClient struct {
	err error
}

/*
rpcFunction  的属性
*/
type rpcFunc struct {
	client *client

	ftyp reflect.Type
	name string

	nout   int
	valOut int
	errOut int

	hasCtx int
	retCh  bool //判断是否是channel
}

func (fn *rpcFunc) processError(err error) []reflect.Value {
	out := make([]reflect.Value, fn.nout)

	if fn.valOut != -1 {
		out[fn.valOut] = reflect.New(fn.ftyp.Out(fn.valOut)).Elem()
	}
	if fn.errOut != -1 {
		out[fn.errOut] = reflect.New(errorType).Elem()
		//	out[fn.errOut].Set(reflect.ValueOf(&ErrClient{err}))
	}

	return out
}

func (c *client) makeOutChan(ctx context.Context, ftyp reflect.Type, valOut int) (func() reflect.Value, makeChanSink) {
	retVal := reflect.Zero(ftyp.Out(valOut))

	chCtor := func() (context.Context, func([]byte, bool)) {
		// unpack chan type to make sure it's reflect.BothDir
		ctyp := reflect.ChanOf(reflect.BothDir, ftyp.Out(valOut).Elem())
		ch := reflect.MakeChan(ctyp, 0) // todo: buffer?
		chCtx, chCancel := context.WithCancel(ctx)
		retVal = ch.Convert(ftyp.Out(valOut))

		buf := (&list.List{}).Init()
		var bufLk sync.Mutex

		return ctx, func(result []byte, ok bool) {
			if !ok {
				chCancel()
				// remote channel closed, close ours too
				ch.Close()
				return
			}

			val := reflect.New(ftyp.Out(valOut).Elem())
			if err := json.Unmarshal(result, val.Interface()); err != nil {
				Log.Errorf("error unmarshaling chan response: %s", err)
				return
			}

			bufLk.Lock()
			if ctx.Err() != nil {
				Log.Errorf("got rpc message with cancelled context: %s", ctx.Err())
				bufLk.Unlock()
				return
			}

			buf.PushBack(val)

			if buf.Len() > 1 {
				Log.Warnw("rpc output message buffer", "n", buf.Len())
				bufLk.Unlock()
				return
			}

			go func() {
				for buf.Len() > 0 {
					front := buf.Front()
					bufLk.Unlock()

					cases := []reflect.SelectCase{
						{
							Dir:  reflect.SelectRecv,
							Chan: reflect.ValueOf(chCtx.Done()),
						},
						{
							Dir:  reflect.SelectSend,
							Chan: ch,
							Send: front.Value.(reflect.Value).Elem(),
						},
					}

					chosen, _, _ := reflect.Select(cases)
					bufLk.Lock()

					switch chosen {
					case 0:
						buf.Init()
					case 1:
						buf.Remove(front)
					}
				}

				bufLk.Unlock()
			}()

		}
	}

	return func() reflect.Value { return retVal }, chCtor
}

func (c *client) sendRequest(ctx context.Context, req request, chCtor makeChanSink) (clientResponse, error) {
	rchan := make(chan clientResponse, 1)
	creq := clientRequest{ //请求的结构体
		req:   req,
		ready: rchan,
		retCh: chCtor,
	}

	//c.requests = make(chan clientRequest) //请求数据的通道

	select { // 已经赋值给了c.requests
	case c.requests <- creq: //赋值就可以在websocket中使用
		//fmt.Printf("received ")
	case <-c.exiting: //是否接收到了值
		return clientResponse{}, fmt.Errorf("websocket routine exiting")
	}

	var ctxDone <-chan struct{}
	var resp clientResponse

	if ctx != nil { //
		ctxDone = ctx.Done()
	}

	// wait for response, handle context cancellation
loop:
	for {
		select {
		case resp = <-rchan:
			break loop
		case <-ctxDone: // send cancel request
			ctxDone = nil

			cancelReq := clientRequest{
				req: request{
					Jsonrpc: "2.0",
					Method:  wsCancel,
					Params:  []Param{{V: reflect.ValueOf(req.ID)}},
					//   Params: []Param{reflect.ValueOf(req.ID)},
				},
			}
			select {
			case c.requests <- cancelReq:
			case <-c.exiting:
				Log.Warn("failed to send request cancellation, websocket routing exited")
			}
		}
	}

	return resp, nil
}

/*
retCh提供了用于处理传入通道消息的上下文和接收器
*/
type makeChanSink func() (context.Context, func([]byte, bool))

/*
用于关闭jsonrpc的方法
*/
type ClientCloser func()
type Address struct{ Str string }

/*
此方法定义了一个jsonrpc2.0的客户端
handler 必须是指向具有函数字段的结构的指针
返回值关闭了客户端的通信
*/
// TODO: Example
func NewClient(addr string, namespace string, handler interface{}, requestHeader http.Header, id address.Address) (ClientCloser, interface{}, error) {
	return NewMergeClient(addr, namespace, []interface{}{handler}, requestHeader, id)
}

func NewMergeClient(addr string, namespace string, outs []interface{}, rerquestHeader http.Header, id address.Address) (ClientCloser, interface{}, error) {
	conn, _, err := websocket.DefaultDialer.Dial(addr, rerquestHeader) //使用websocket的DialHttp的方法可以加headers

	if err != nil {
		log.Fatal("dial:", err)
	}

	stop := make(chan struct{}) //停止的通道
	//初始化client

	c := client{}
	exiting := make(chan struct{})             //接收数据的通道
	c.requests = make(chan clientRequest, 100) //请求数据的通道
	c.exiting = exiting
	c.namespace = namespace

	//处理函数的map类型的rpc  key是函数  rpcHandler是处理函数得到数据
	handlers := map[string]rpcHandler{}

	var data interface{}
	go (&wsConn{ //开启多个websocket的连接线程     websocket的方法类
		conn:     conn, //已经建立了连接
		handler:  handlers,
		requests: c.requests,
		stop:     stop,
		exiting:  exiting,
	}).handleWsConn(context.TODO()) //和下面的是并发进行的 下面通过通道进行传输

	for _, handler := range outs {
		htyp := reflect.TypeOf(handler)
		if htyp.Kind() != reflect.Ptr {
			return nil, nil, fmt.Errorf("expected handler to be a pointer")
		}
		typ := htyp.Elem() //取元素
		if typ.Kind() != reflect.Struct {
			return nil, nil, fmt.Errorf("handler should be a struct")
		}
		val := reflect.ValueOf(handler) //得到函数

		//遍历struct中的元素
		for i := 0; i < typ.NumField(); i++ {
			fn, err := c.makeRpcFunc(typ.Field(i)) //对每一个元素进行一个制作rpc函数的查找方法
			if err != nil {
				return nil, nil, err
			}
			val.Elem().Field(i).Set(fn) //fn是得到的值, 将数值赋值给val
			//val.Elem().MethodByName(typ.Field(i).Name)
			paramList := []reflect.Value{
				reflect.ValueOf(context.Background()),
			}
			//	fmt.Println(reflect.ValueOf(id))
			if typ.Field(i).Type.NumIn() > 1 {
				paramList = append(paramList, reflect.ValueOf(id))
				//	fmt.Println("aaaa")
			}
			var temp *type_.TipSet
			if typ.Field(i).Type.NumIn() > 2 {
				paramList = append(paramList, reflect.ValueOf(temp)) // reflect.ValueOf(temp))
			}
			//	val.Elem().Field(i).Type().Method(paramList)
			totalValue := val.Elem().Field(i).Call(paramList)
			if !totalValue[1].IsNil() {
				Log.Errorf(": %s", totalValue[1])
			} else {
				data = totalValue[0].Interface()
			}

			//val.MethodByName(typ.Field(i).Name).Call(paramList)
			//将函数指针参数指向通用算法函数
			//*handle()
			//val.Elem().Field(i)
		}
	}

	return func() {
		close(stop)
		<-exiting
	}, data, nil
}

/*
type rpcFunc struct {
	client *client

	ftyp reflect.Type  //函数的type
	name string

	nout   int
	valOut int
	errOut int

	hasCtx int  是否含有context类型
	retCh  bool  //判断是否为channel
}
*/
func (c *client) makeRpcFunc(f reflect.StructField) (reflect.Value, error) {
	ftyp := f.Type
	//判断结构体中是否为1个函数
	if ftyp.Kind() != reflect.Func {
		return reflect.Value{}, fmt.Errorf("handler field not a func")
	}
	//函数的rpc
	fun := &rpcFunc{
		client: c,    // 引用一个client
		ftyp:   ftyp, //function的type
		name:   f.Name,
	}
	fun.valOut, fun.errOut, fun.nout = processFuncOut(ftyp) //给三个属性赋值   此方法查找函数的值的位置以及错误
	// 函数返回的值的个数
	//valOut = 0   在第0个位置   errOut = 1   输出的返回值的个数为n 2

	if ftyp.NumIn() > 0 && ftyp.In(0) == contextType { //参数
		fun.hasCtx = 1 // 走到这
	}
	//FALSE  判断是否是channel
	fun.retCh = fun.valOut != -1 && ftyp.Out(fun.valOut).Kind() == reflect.Chan //判断是否为channel
	//将函数指针参数指向通用算法函数
	return reflect.MakeFunc(ftyp, fun.handleRpcCall), nil //  制作函数，第一个参数是reflect.TypeOf,第二个参数是reflect.ValueOf
}

/*
   处理RPC 的call的方法
*/
func (fn *rpcFunc) handleRpcCall(args []reflect.Value) (results []reflect.Value) {
	id := atomic.AddInt64(&fn.client.idCtr, 1)   //取得函数得参数
	params := make([]Param, len(args)-fn.hasCtx) //用所有的参数减去 context的参数
	for i, arg := range args[fn.hasCtx:] {       //context 没有参数
		params[i] = Param{
			V: arg,
			//data: arg.Index(i),
			//Data: []byte("adsafsafsafafgdsf"),
		}
		temp, _ := params[i].MarshalJson()
		params[i].UnmarshalJson(temp)
		//params[i].data = []byte("adsafsafsafafgdsf")
		//	temp1, _ := json.Marshal(params[i].v.Interface())		//	temp1, _ := json.Marshal(params[i].v.Interface())
		//	fmt.Println("aaaaa", arg, params[i].Data)
	} //观察是否有ctx全文内容
	/*	if cap(params) != 0 {
		params = params[0:1]
	}*/

	var ctx context.Context
	var span *trace.Span //追踪得span
	if fn.hasCtx == 1 {
		ctx = args[0].Interface().(context.Context) //ctx 赋值
		ctx, span = trace.StartSpan(ctx, "api.call")
		defer span.End()
	}

	retVal := func() reflect.Value { return reflect.Value{} }

	// if the function returns a channel, we need to provide a sink for the
	// messages
	var chCtor makeChanSink
	if fn.retCh {
		retVal, chCtor = fn.client.makeOutChan(ctx, fn.ftyp, fn.valOut)
	}
	req := request{ //进行请求得提交
		Jsonrpc: "2.0",
		ID:      &id,
		Method:  fn.client.namespace + "." + fn.name,
		Params:  params, //可以接收到参数
	}

	if span != nil {
		span.AddAttributes(trace.StringAttribute("method", req.Method))

		eSC := base64.StdEncoding.EncodeToString(
			propagation.Binary(span.SpanContext()))
		req.Meta = map[string]string{
			"SpanContext": eSC,
		}
	}

	resp, err := fn.client.sendRequest(ctx, req, chCtor) //发送请求
	//     ------------------------ response的值

	if err != nil {
		return fn.processError(fmt.Errorf("sendRequest failed: %s", err))
	}

	if *resp.ID != *req.ID {
		return fn.processError(fmt.Errorf("request and response id didn't match"))
	}

	if fn.valOut != -1 && !fn.retCh { //有返回值并且不是channel的情况下
		val := reflect.New(fn.ftyp.Out(fn.valOut))

		if resp.Result != nil {
			Log.Debugw("rpc result", "type", fn.ftyp.Out(fn.valOut))
			if err := json.Unmarshal(resp.Result, val.Interface()); err != nil {
				Log.Warnw("unmarshaling failed", "message", string(resp.Result))
				return fn.processError(fmt.Errorf("unmarshaling result: %s", err))
			}
		}
		retVal = func() reflect.Value { return val.Elem() }
	}

	return fn.processResponse(resp, retVal()) //返回返回的值 ,然后进行处理
}

func (fn *rpcFunc) processResponse(resp clientResponse, rval reflect.Value) []reflect.Value {
	out := make([]reflect.Value, fn.nout)

	if fn.valOut != -1 {
		out[fn.valOut] = rval
	}
	if fn.errOut != -1 {
		out[fn.errOut] = reflect.New(errorType).Elem()
		if resp.Error != nil {
			out[fn.errOut].Set(reflect.ValueOf(resp.Error))
		}
	}
	return out
}
