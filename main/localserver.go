package main

import (
	"context"
	"fmt"
	"github.com/filecoin-project/lotus/chain/address"
	type_ "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/addrutil"
	cid "github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"math/big"
	http_ "yuncun.com/main/httptest"
	server_ "yuncun.com/main/rpc"
)

type SimpleServerHandler struct {
}

func (h *SimpleServerHandler) NetAddrsListen(context context.Context) (peer.AddrInfo, error) {
	//info := &peer.AddrInfo{}
	addrs := []string{"/ip4/127.0.0.1/tcp/12000/ipfs/12D3KooWH7GXgKS4gshpFrbYQ6UxzHeJxfqz5FyfhCn2RnNjofph", "/ip4/192.168.0.10/tcp/12000/ipfs/12D3KooWH7GXgKS4gshpFrbYQ6UxzHeJxfqz5FyfhCn2RnNjofph", "/ip6/::1/tcp/12000/ipfs/12D3KooWH7GXgKS4gshpFrbYQ6UxzHeJxfqz5FyfhCn2RnNjofph"}
	addrinfo, _ := addrutil.ParseAddresses(context, addrs)
	return addrinfo[0], nil
}
func (h *SimpleServerHandler) ID(context.Context) (string, error) {
	id := "12D3KooWH7GXgKS4gshpFrbYQ6UxzHeJxfqz5FyfhCn2RnNjofph"
	return id, nil
}
func (h *SimpleServerHandler) ChainHead(context.Context) (*type_.TipSet, error) {
	cidall := []cid.Cid{}
	cid1, _ := cid.Parse("/ip4/127.0.0.1/tcp/12000/ipfs/QmSsw6EcnwEiTT9c4rnAGeSENvsJMepNHmbrgi2S9bXNJr")
	cidall = append(cidall, cid1)
	address, _ := address.NewFromString("t01007")
	postticket := []type_.EPostTicket{{[]byte("s1badzObR91BWnQcN+mDyOlqN6wt2WXwnEx3f8toXgM="), 7, 1}, {[]byte("80GuH7JrQc03PSrUXPIf9iQQbshxFmbtJhWEc2iiOhM="), 197, 8}}
	eposproof := type_.EPostProof{[]byte("izkyTqwnOj9G72nFdy5j8jIOKtSdcc4s3zLgEnibi8BDfwJpTUYqieGG2TnOQpgKkUSkUcsuCAufPfHHIF7Vo9Fle0/PMsu1R23OkBPgbMQNwaXR1h6JeOWy84yMXiB1BDOs+B2ykNf7iUk/2AAuI9wRPHbOCXds4hqMFSjtYDEqW4XSkxymEdkHYvEORuVAsFE2ySI+VGeW251952XvsWKlH4xwWaQWuNikJx0uJAUD6Gvc5YGp0rqhUhL6Uh93tNBtOJUCoIiWc3+ZTVaZgITt3zbCmVRXGHl5yZfGs2vQIVQ9bUXjTb9FMhVelGP4k4wZiz6n+iRdZQJGN1YXDOnaAxZlT/JEHZBMn/kx1qTwzKHIrJ9WBvucY9v0myH4Dh9BMnZ88OvMx4IpIZVLDiQqAj1AKtRz4s6HGe9kL1auwTxYn4vwwmtnoi9kuHFpsHC7wxDgV79qbz2Q1zzw4JG7zh6C82py0t4CXgjJuFSdzJttzHTK4MP0G3W0Ffqe"),
		[]byte("gp78qhderT8/sysejATqTloaWWAmUIfIg8b/Jygwk3HaeZcxW7s0kr4D9Exs1z2MDSnFhDZ9NZqBSaZ+vACdcaBDAacVFILukm1BzTRk2DryiG189nH2zex/fNQh08TU"), postticket}
	ticket := &type_.Ticket{VRFProof: []byte("ktih2D/5O5zdvwB1a/4NfXQISibiLwPtW+Efxo4U1PlQexYLs4mFldQXtS4EIAzZDCQ11nCuWHN6Xgng7GYpUC4aGPIkbai+1JjA8adcSPBhuDr+U6UGIJ/kc2Kp6daI")}
	blsaggregate := type_.Signature{
		Type: "bls",
		Data: nil,
	}
	blks := []*type_.BlockHeader{{
		Miner:                 address,
		Ticket:                ticket,
		EPostProof:            eposproof,
		ParentWeight:          type_.BigInt{big.NewInt(596518205)},
		Height:                51174,
		ParentStateRoot:       cid1,
		ParentMessageReceipts: cid1,
		Messages:              cid1,
		BLSAggregate:          blsaggregate,
		Parents:               cidall,
	}}
	tipset, _ := type_.NewTipSet(blks)
	fmt.Println(tipset.Height())
	return tipset, nil
}

//{"jsonrpc":"2.0","result":{"MinerPower":"0","TotalPower":"633825300114114756364956860416"},"id":3}
func (h *SimpleServerHandler) StateMinerPower(context.Context, address.Address, *type_.TipSet) (server_.MinerPower, error) {
	temp, _ := new(big.Int).SetString("633825300114114756364956860416", 10)
	minerpower := server_.MinerPower{
		MinerPower: type_.NewInt(1000),
		TotalPower: type_.BigInt{temp},
	}
	return minerpower, nil
}
func (h *SimpleServerHandler) WalletBalance(context.Context, address.Address) (type_.BigInt, error) {
	temp, _ := new(big.Int).SetString("487776486641635185", 10)
	return type_.BigInt(type_.BigInt{temp}), nil
}
func (h *SimpleServerHandler) StateMinerSectorSize(context.Context, address.Address, *type_.TipSet) (uint64, error) {
	return 1073741824, nil
}
func (h *SimpleServerHandler) StateMinerSectorCount(context.Context, address.Address, *type_.TipSet) (server_.MinerSectors, error) {
	minersector := server_.MinerSectors{
		Pset: 49,
		Sset: 261,
	}
	return minersector, nil
}
func (h *SimpleServerHandler) ActorAddress(context.Context) (string, error) {
	return "t06266", nil
}

//sever 端进行赋值
func main() {
	addr := address.Address{}
	addr, _ = address.NewFromString("t06266")
	serverHandler := &SimpleServerHandler{}
	id, _ := serverHandler.ID(context.Background())
	ip, _ := serverHandler.NetAddrsListen(context.Background())
	height, _ := serverHandler.ChainHead(context.Background())
	minerpower, _ := serverHandler.StateMinerPower(context.Background(), addr, nil)
	balance, _ := serverHandler.WalletBalance(context.Background(), addr)
	sectorsize, _ := serverHandler.StateMinerSectorSize(context.Background(), addr, nil)
	sectorcount, _ := serverHandler.StateMinerSectorCount(context.Background(), addr, nil)
	actor, _ := serverHandler.ActorAddress(context.Background())
	fmt.Println(id, ip, height, minerpower, balance, sectorsize, sectorcount, actor)
	rpcServer := server_.NewServer()
	rpcServer.Register("Filecoin", serverHandler)

	server := http_.NewServer(rpcServer)
	server.Start()
	defer server.Close()
}
