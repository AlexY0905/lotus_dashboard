package rpc

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/opentracing/opentracing-go/log"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
	"golang.org/x/xerrors"
	"io"
	"reflect"
)

//定义一些常量的错误

type request struct {
	Jsonrpc string            `json:"jsonrpc"`
	ID      *int64            `json:"id,omitempty"`
	Method  string            `json:"method"`
	Params  []Param           `json:"params,omitempty"` //配置的parmeter的类的参数
	Meta    map[string]string `json:"meta,omitempty"`
}
type respError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcHandler struct {
	paramReceivers []reflect.Type //接收的类型值
	nParams        int            //接收的个数

	receiver    reflect.Value //接收者的值
	handlerFunc reflect.Value //处理的函数的值

	hasCtx int

	errOut int
	valOut int
}

type response struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	ID      int64       `json:"id"`
	Error   *respError  `json:"error,omitempty"`
}

type handlers map[string]rpcHandler //string指的是方法名字

type rpcErrFunc func(w func(func(writer io.Writer)), req *request, code int, err error)
type chanOut func(reflect.Value) interface{}

func (e *respError) Error() string {
	if e.Code >= -32768 && e.Code <= -32000 {
		return fmt.Sprintf("RPC error #{e.Code}:#{e.Message}")
	}
	return e.Message
}

//得到trace的span  跟踪整个行为链条
func (h handlers) getSpan(ctx context.Context, req request) (context.Context, *trace.Span) {
	if req.Meta == nil {
		return ctx, nil
	}
	if esc, ok := req.Meta["SpanContext"]; ok {
		bsc := make([]byte, base64.StdEncoding.DecodedLen(len(esc))) //创建一个bsc的slice,将esc赋值给bsc
		_, err := base64.StdEncoding.Decode(bsc, []byte(esc))
		if err != nil {
			Log.Errorf("SpanContext: decode", "error", err)
			return ctx, nil
		}
		//If b has an unsupported version ID or contains no TraceID, FromBinary
		//// returns with ok==false. 判断bsc中是否有正规的trace id
		sc, ok := propagation.FromBinary(bsc)
		if !ok {
			Log.Errorf("SpanContext: could not create span", "data", bsc)
			return ctx, nil
		}
		ctx, span := trace.StartSpanWithRemoteParent(ctx, "api.handle", sc)
		span.AddAttributes(trace.StringAttribute("method", req.Method)) //給span添加方法追蹤
		return ctx, span
	}
	return ctx, nil
}

//调用参数的方式

func doCall(methodName string, f reflect.Value, params []reflect.Value) (out []reflect.Value, err error) {
	defer func() {
		if i := recover(); i != nil {
			err = fmt.Errorf("panic in rpc method '%s': %s", methodName, i)
			log.Error(err)
		}
	}()

	out = f.Call(params)
	return out, nil
}

//属于handlers  map[string]rpcHandler的方法

func (h handlers) handle(ctx context.Context, req request, w func(func(at io.Writer)), rpcError rpcErrFunc, done func(keepCtx bool), chOut chanOut) {
	ctx, span := h.getSpan(ctx, req)
	defer span.End()

	////type handlers map[string]rpcHandler  //string指的是方法名字
	handler, ok := h[req.Method] //rpcHandler的结构体
	if !ok {
		rpcError(w, &req, rpcMethodNotFound, fmt.Errorf("method '%s' not found", req.Method))
		done(false)
		return
	}
	//判断请求的参数的个数
	if len(req.Params) != handler.nParams {
		rpcError(w, &req, rpcInvalidParams, fmt.Errorf("wrong param count"))
		done(false)
		return
	}
	//handlerFunc reflect.Value //处理的函数的值  判断handler的类型是否为channel
	outChannel := handler.valOut != -1 && handler.handlerFunc.Type().Out(handler.valOut).Kind() == reflect.Chan
	defer done(outChannel)

	//判断方法是否存在rpcMethodNotFound， no out channel support
	if chOut == nil && outChannel {
		rpcError(w, &req, rpcMethodNotFound, fmt.Errorf("method '%s' not supported in this mode (no out channel support)", req.Method))
		return
	}

	callParams := make([]reflect.Value, 1+handler.hasCtx+handler.nParams) //创建一个调用参数param的slice
	callParams[0] = handler.receiver                                      //接收者  receiver    reflect.Value //接收者的值

	if handler.hasCtx == 1 {
		callParams[1] = reflect.ValueOf(ctx)
	}

	//取得调用的参数
	//paramReceivers []reflect.Type //接收的类型值
	for i := 0; i < handler.nParams; i++ {
		rp := reflect.New(handler.paramReceivers[i])
		if err := json.NewDecoder(bytes.NewReader(req.Params[i].Data)).Decode(rp.Interface()); err != nil {
			rpcError(w, &req, rpcParseError, fmt.Errorf("unmarshaling params for '%s': %w", handler.handlerFunc, err))
			return
		}
		callParams[i+1+handler.hasCtx] = reflect.ValueOf(rp.Elem().Interface())
	}

	//进行调用  返回类型  out []reflect.Value, err error
	callResult, err := doCall(req.Method, handler.handlerFunc, callParams)
	if err != nil {
		rpcError(w, &req, 0, fmt.Errorf("fatal error calling '%s': %w", req.Method, err))
		return
	}
	////////////////返回结果

	//对response 的 resp进行赋值
	resp := response{
		Jsonrpc: "2.0",
		ID:      *req.ID,
	}

	if handler.errOut != -1 {
		err := callResult[handler.errOut].Interface()
		if err != nil {
			Log.Warnf("error in RPC call to '%s': %+v", req.Method, err)
			resp.Error = &respError{
				Code:    1,
				Message: err.(error).Error(),
			}
		}
	}
	if handler.valOut != -1 {
		resp.Result = callResult[handler.valOut].Interface()
	}

	w(func(w io.Writer) {
		if resp.Result != nil && reflect.TypeOf(resp.Result).Kind() == reflect.Chan {
			resp.Result = chOut(callResult[handler.valOut])
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error(err)
			return
		}
	})
}

// Register

func (h handlers) register(namespace string, r interface{}) {
	val := reflect.ValueOf(r)
	//TODO: expect ptr
	//	fmt.Println(val)
	for i := 0; i < val.NumMethod(); i++ {
		method := val.Type().Method(i)

		funcType := method.Func.Type()
		hasCtx := 0
		if funcType.NumIn() >= 2 && funcType.In(1) == contextType {
			hasCtx = 1
		}

		ins := funcType.NumIn() - 1 - hasCtx
		recvs := make([]reflect.Type, ins)
		for i := 0; i < ins; i++ {
			recvs[i] = method.Type.In(i + 1 + hasCtx)
		}

		valOut, errOut, _ := processFuncOut(funcType)

		h[namespace+"."+method.Name] = rpcHandler{
			paramReceivers: recvs,
			nParams:        ins,

			handlerFunc: method.Func,
			receiver:    val,

			hasCtx: hasCtx,

			errOut: errOut,
			valOut: valOut,
		}
		//	fmt.Println(h[namespace+"."+method.Name].handlerFunc)
	}
}

func (h handlers) handleReader(ctx context.Context, r io.Reader, w io.Writer, rpcError rpcErrFunc) {
	wf := func(cb func(io.Writer)) {
		cb(w)
	}

	var req request
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		rpcError(wf, &req, rpcParseError, xerrors.Errorf("unmarshaling request: %w", err))
		return
	}
	h.handle(ctx, req, wf, rpcError, func(bool) {}, nil)
}
