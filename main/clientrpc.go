package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	_ "github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/address"
	type_ "github.com/filecoin-project/lotus/chain/types"
	"github.com/libp2p/go-libp2p-core/peer"
	mytype_ "go/types"
	"math/big"
	"strings"
	"time"
	server "yuncun.com/main/server"
	//	"math/big"
	"net/http"
	"os/exec"
	//"strings"
	//"time"
	client_ "yuncun.com/main/rpc"
)

type null mytype_.Nil
type SimpleClientHandler struct {
}

//                  1234  和  2345

/*
curl -X POST      -H "Content-Type: application/json"
--data '{ "jsonrpc": "2.0", "method": "Filecoin.NetAddrsListen", "params": [], "id": 3 }'      'http://127.0.0.1:1234/rpc/v0'
{"jsonrpc":"2.0","result":{"Addrs":["/ip4/127.0.0.1/tcp/12000","/ip4/192.168.0.10/tcp/12000","/ip6/::1/tcp/12000"],"ID":"12D3KooWH7GXgKS4gshpFrbYQ6UxzHeJxfqz5FyfhCn2RnNjofph"},"id":3}
*/
func (h *SimpleClientHandler) GetIP() json.RawMessage {
	return nil
}

/*
得到 GPU信息
*/
func (h *SimpleClientHandler) GetGPU() string {
	cmd := exec.Command("/bin/bash", "-c", ` nvidia-smi -L`)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error:can not obtain stdout pipe for command:%s\n", err)
		return ""
	}
	//执行命令
	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err,", err)
		return ""
	}
	//使用带缓冲的读取器
	outputBuf := bufio.NewReader(stdout)
	for {
		//一次获取一行,_ 获取当前行是否被读完
		output, _, err := outputBuf.ReadLine()
		if err != nil {

			// 判断是否到文件的结尾了否则出错
			if err.Error() != "EOF" {
				fmt.Printf("Error :%s\n", err)
			}
			return ""
		}
		return string(output)
		fmt.Printf("%s\n", string(output))
	}
	//wait 方法会一直阻塞到其所属的命令完全运行结束为止
	if err := cmd.Wait(); err != nil {
		fmt.Println("wait:", err.Error())
		return ""
	}

	return ""
}

/*
得到CPU信息  查看逻辑CPU  cat /proc/cpuinfo |grep "processor"|sort -u|wc -l
32
*/
func (h *SimpleClientHandler) GetCPU() string {
	cmd := exec.Command("/bin/bash", "-c", `cat /proc/cpuinfo |grep "processor"|sort -u|wc -l`)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error:can not obtain stdout pipe for command:%s\n", err)
		return ""
	}
	//执行命令
	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err,", err)
		return ""
	}
	//使用带缓冲的读取器
	outputBuf := bufio.NewReader(stdout)
	for {
		//一次获取一行,_ 获取当前行是否被读完
		output, _, err := outputBuf.ReadLine()
		if err != nil {

			// 判断是否到文件的结尾了否则出错
			if err.Error() != "EOF" {
				fmt.Printf("Error :%s\n", err)
			}
			return ""
		}
		return string(output)
		fmt.Printf("%s\n", string(output))
	}
	//wait 方法会一直阻塞到其所属的命令完全运行结束为止
	if err := cmd.Wait(); err != nil {
		fmt.Println("wait:", err.Error())
		return ""
	}

	return ""
}

/*
得到内存信息  cat /proc/meminfo |grep "MemTotal"
MemTotal:       131958632 kB
*/
func (h *SimpleClientHandler) GetMem() string {
	cmd := exec.Command("/bin/bash", "-c", `cat /proc/meminfo |grep "MemTotal"`)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error:can not obtain stdout pipe for command:%s\n", err)
		return ""
	}
	//执行命令
	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err,", err)
		return ""
	}
	//使用带缓冲的读取器
	outputBuf := bufio.NewReader(stdout)
	for {
		//一次获取一行,_ 获取当前行是否被读完
		output, _, err := outputBuf.ReadLine()
		if err != nil {

			// 判断是否到文件的结尾了否则出错
			if err.Error() != "EOF" {
				fmt.Printf("Error :%s\n", err)
			}
			return ""
		}
		return string(output)
		fmt.Printf("%s\n", string(output))
	}
	//wait 方法会一直阻塞到其所属的命令完全运行结束为止
	if err := cmd.Wait(); err != nil {
		fmt.Println("wait:", err.Error())
		return ""
	}

	return ""
}

func (h *SimpleClientHandler) GetTOKEN() string {
	cmd := exec.Command("/bin/bash", "-c", `cat ~/.lotus/token`)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error:can not obtain stdout pipe for command:%s\n", err)
		return ""
	}
	//执行命令
	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err,", err)
		return ""
	}
	//使用带缓冲的读取器
	outputBuf := bufio.NewReader(stdout)
	for {
		//一次获取一行,_ 获取当前行是否被读完
		output, _, err := outputBuf.ReadLine()
		if err != nil {

			// 判断是否到文件的结尾了否则出错
			if err.Error() != "EOF" {
				fmt.Printf("Error :%s\n", err)
			}
			return ""
		}
		return string(output)
		fmt.Printf("%s\n", string(output))
	}
	//wait 方法会一直阻塞到其所属的命令完全运行结束为止
	if err := cmd.Wait(); err != nil {
		fmt.Println("wait:", err.Error())
		return ""
	}

	return ""
}

/*
curl -X POST      -H "Content-Type: application/json"
--data '{ "jsonrpc": "2.0", "method": "Filecoin.ID", "params": [], "id": 3 }'      'http://127.0.0.1:1234/rpc/v0'
{"jsonrpc":"2.0","result":"12D3KooWH7GXgKS4gshpFrbYQ6UxzHeJxfqz5FyfhCn2RnNjofph","id":3}

curl -X POST      -H "Content-Type: application/json"       -H "Authorization: Bearer $(cat ~/.lotus/token)"   --data '{ "jsonrpc": "2.0", "method": "Filecoin.StateMinerPeerID", "params": ["t05671", null], "id": 3 }'      'http://127.0.0.1:1234/rpc/v0'
{"jsonrpc":"2.0","result":"12D3KooWHC8pcFTBFbg4CKvqFhK7Ag3Vb6uD25Ejzry5Rz59BLX1","id":3}
ID
*/
func (h *SimpleClientHandler) GetID() json.RawMessage {
	return nil
}

/*
curl -X POST      -H "Content-Type: application/json" --data '{ "jsonrpc": "2.0", "method": "Filecoin.ActorAddress", "params": [], "id": 3 }'      'http://127.0.0.1:2345/rpc/v0'

{"jsonrpc":"2.0","result":"t05671","id":3}
名字
*/
func (h *SimpleClientHandler) GetActorAddress() json.RawMessage {
	return nil
}

/*
同步高度
*/

/*
curl -X POST \
     -H "Content-Type: application/json" \
     --data '{ "jsonrpc": "2.0", "method": "Filecoin.ChainHead", "params": [], "id": 3 }' \
     'http://127.0.0.1:1234/rpc/v0'

得到高度

*/
func (h *SimpleClientHandler) GetChainHead() json.RawMessage {
	return nil
}

/*
curl -X POST -H "Content-Type: application/json" --data '{ "jsonrpc": "2.0", "method": "Filecoin.StateMinerPower", "params": ["t0101", null], "id": 3 }'  'http://127.0.0.1:1234/rpc/v0'
算力
{"jsonrpc":"2.0","result":{"MinerPower":"0","TotalPower":"633825300114114756364956860416"},"id":3}

*/
func (h *SimpleClientHandler) GetMinerPower() json.RawMessage {
	return nil
}

/*
curl -X POST      -H "Content-Type: application/json"       -H "Authorization: Bearer $(cat ~/.lotus/token)" --data '{ "jsonrpc": "2.0", "method": "Filecoin.WalletBalance", "params": ["t05671"], "id": 3 }'      'http://127.0.0.1:1234/rpc/v0'
{"jsonrpc":"2.0","result":"487546486641635185","id":3}
*/
func (h *SimpleClientHandler) GetWalletBalance() json.RawMessage {
	return nil
}

/*
curl -X POST      -H "Content-Type: application/json"       -H "Authorization: Bearer $(cat ~/.lotus/token)"
--data '{ "jsonrpc": "2.0", "method": "Filecoin.StateMinerSectorSize", "params": ["t05671", null], "id": 3 }'      'http://127.0.0.1:1234/rpc/v0'
{"jsonrpc":"2.0","result":1073741824,"id":3}   1G左右
*/
func (h *SimpleClientHandler) GetSectorSize() json.RawMessage {
	return nil
}

/*
得到区块的size
 curl -X POST      -H "Content-Type: application/json"       -H "Authorization: Bearer $(cat ~/.lotus/token)"
--data '{ "jsonrpc": "2.0", "method": "Filecoin.StateMinerSectorCount", "params": ["t05671", null], "id": 3 }'      'http://127.0.0.1:1234/rpc/v0'
{"jsonrpc":"2.0","result":{"Pset":49,"Sset":261},"id":3}
*/
func (h *SimpleClientHandler) GetSectorCount() json.RawMessage {
	return nil
}

/*
挖矿的个数
*/
func (h *SimpleClientHandler) GetMinedCount() json.RawMessage {
	return nil
}

//MinerPower

func main() {
	lotus := &server.Lotus{
		LotusNetAddrsListenData:   "",
		LotusIdData:               "",
		LotusChainHeadData:        0,
		MinerData:                 "",
		LotusStateMinerPowerData:  type_.BigInt{},
		LotusWalletBalanceData:    type_.BigInt{},
		LotusStateMinerSectorSize: 0,
		CPU:                       "",
		GPU:                       "",
		Mem:                       "",
	}
	//	client := &SimpleClientHandler{}
	//CPU := client.GetCPU()
	lotus.CPU = "24"
	//GPU := client.GetGPU()
	lotus.GPU = "INVIDIA 2040"
	//Mem := client.GetMem()
	lotus.Mem = "128"

	/*
		在 1234 监听
	*/
	var lotusNetAddrsListen struct {
		NetAddrsListen func(context.Context) (peer.AddrInfo, error)
	}
	var lotusId struct { //后面改
		ID func(context.Context) (string, error)
	}
	var lotusChainHead struct {
		ChainHead func(context.Context) (*type_.TipSet, error)
	}
	var lotusStateMinerPower struct {
		//StateMinerPower func(context.Context, address.Address, *type_.TipSet) (MinerPower, error)
		StateMinerPower func(context.Context, address.Address, *type_.TipSet) (client_.MinerPower, error)
	}
	var lotusWalletBalance struct {
		WalletBalance func(context.Context, address.Address) (type_.BigInt, error)
	}

	var lotusStateMinerSectorSize struct {
		StateMinerSectorSize func(context.Context, address.Address, *type_.TipSet) (uint64, error)
	}
	var lotusStateMinerSectorCount struct {
		StateMinerSectorCount func(context.Context, address.Address, *type_.TipSet) (client_.MinerSectors, error)
	}
	//	在2345 监听

	var miner struct {
		ActorAddress func(context.Context) (string, error)
	}
	lotusrequest := http.Header{}
	/*
		Content-Type:  "application/json",
				Authorization: "Bearer $(cat ~/.lotus/token)"
	*/
	//	u := url.URL{Scheme: "ws", Host: "127.0.0.1:1234", Path: "/rpc/v0"}
	lotusrequest.Add("Content-Type", "application/json")
	//lotusrequest.Add("Authorization", "Bearer "+client.GetTOKEN())
	minerrequest := http.Header{}
	minerrequest.Add("Content-Type", "application/json")
	addr := address.Address{}
	for {
		lotusrpcClient1, lotusNetAddrsListenData, clienterr1 := client_.NewClient("ws://127.0.0.1:12666", "Filecoin", &lotusNetAddrsListen, lotusrequest, addr)
		if clienterr1 == nil {
			if lotusNetAddrsListenData == nil {
				lotus.LotusNetAddrsListenData = ""
			} else {
				//fmt.Println(strings.Split(lotusNetAddrsListenData.(peer.AddrInfo).Addrs[1].String(), "/")[2])
				lotus.LotusNetAddrsListenData = strings.Split(lotusNetAddrsListenData.(peer.AddrInfo).Addrs[1].String(), "/")[2]
			}
		} else {
			lotus.LotusNetAddrsListenData = ""
		}
		lotusrpcClient2, lotusIdData, clienterr2 := client_.NewClient("ws://127.0.0.1:12666", "Filecoin", &lotusId, lotusrequest, addr) //
		if clienterr2 == nil {
			lotus.LotusIdData = lotusIdData.(string)
		} else {
			lotus.LotusIdData = "没有识别id"
		}
		lotusrpcClient3, lotusChainHeadData, clienterr3 := client_.NewClient("ws://127.0.0.1:12666", "Filecoin", &lotusChainHead, lotusrequest, addr)
		if clienterr3 == nil && lotusChainHeadData != nil { //需要在看看
			lotus.LotusChainHeadData = lotusChainHeadData.(*type_.TipSet).Height()
		} else {
			lotus.LotusChainHeadData = 0.0
		}
		minerrpcClient, minerData, err := client_.NewClient("ws://127.0.0.1:12666", "Filecoin", &miner, minerrequest, addr) //defer lotusrpcClient1()
		if err == nil {
			addr, _ = address.NewFromString(minerData.(string))
			lotus.MinerData = minerData.(string)
			lotusrpcClient4, lotusStateMinerPowerData, clienterr4 := client_.NewClient("ws://127.0.0.1:12666", "Filecoin", &lotusStateMinerPower, lotusrequest, addr)
			if clienterr4 == nil && lotusStateMinerPowerData != nil {
				lotus.LotusStateMinerPowerData = lotusStateMinerPowerData.(client_.MinerPower).MinerPower
			} else {
				lotus.LotusStateMinerPowerData = type_.BigInt{big.NewInt(0)}
			}
			lotusrpcClient5, lotusWalletBalanceData, clienterr5 := client_.NewClient("ws://127.0.0.1:12666", "Filecoin", &lotusWalletBalance, lotusrequest, addr)
			if clienterr5 == nil && lotusWalletBalanceData != nil {
				lotus.LotusWalletBalanceData = lotusWalletBalanceData.(type_.BigInt)
			} else {
				lotus.LotusWalletBalanceData = type_.BigInt{big.NewInt(0)}
			}
			lotusrpcClient6, lotusStateMinerSectorSizeData, clienterr6 := client_.NewClient("ws://127.0.0.1:12666", "Filecoin", &lotusStateMinerSectorSize, lotusrequest, addr)
			lotusrpcClient7, lotusStateMinerSectorCountData, clienterr7 := client_.NewClient("ws://127.0.0.1:12666", "Filecoin", &lotusStateMinerSectorCount, lotusrequest, addr)
			if clienterr6 == nil && clienterr7 == nil {
				lotusStateMinerSectorSizeData := lotusStateMinerSectorSizeData.(uint64) * lotusStateMinerSectorCountData.(client_.MinerSectors).Pset
				lotus.LotusStateMinerSectorSize = lotusStateMinerSectorSizeData
			} else {
				lotus.LotusStateMinerSectorSize = 0
			}
			lotusrpcClient4()
			lotusrpcClient5()
			lotusrpcClient6()
			lotusrpcClient7()
		}
		lotusrpcClient1()
		lotusrpcClient2()
		lotusrpcClient3()
		minerrpcClient()
		fmt.Print(lotus)
		request := http.Header{}
		server.NewTransmitClient("http://127.0.0.1:10080/local/get_local", request, lotus)
		time.Sleep(1 * time.Minute)
	}

	//request.Add("Content-Type", "application/json")
	//go
}
