package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

type Param struct {
	Data []byte        //从unmarshal的数据
	V    reflect.Value // 去编码
}

var (
	errorType   = reflect.TypeOf(new(error)).Elem()
	contextType = reflect.TypeOf(new(context.Context)).Elem()
)

/*
将json字符串解析成对应的数据结构
*/
func (p *Param) UnmarshalJson(raw []byte) error {
	p.Data = make([]byte, len(raw))
	copy(p.Data, raw)
	return nil
}

/*
将数据编码成json字符串
*/
func (p *Param) MarshalJson() ([]byte, error) {
	return json.Marshal(p.V.Interface())
}

/*
 此方法查找函数的值以及错误
*/
func processFuncOut(funcType reflect.Type) (valOut int, errOut int, n int) {
	errOut = -1 // -1 if not found  没有找到就赋值给 -1
	valOut = -1
	n = funcType.NumOut() //NumOut返回函数类型的输出参数计数。

	switch n {
	case 0:
	case 1:
		if funcType.Out(0) == errorType {
			errOut = 0
		} else {
			valOut = 0
		}
	case 2: // 输出的是两个返回值
		valOut = 0
		errOut = 1
		if funcType.Out(1) != errorType {
			panic("expected error as second return value")
		}
	default:
		errstr := fmt.Sprintf("too many return values: %s", funcType)
		panic(errstr)
	}

	return
}
