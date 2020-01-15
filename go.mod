module main

go 1.13

require (
	//	github.com/felixge/tcpkeepalive v0.0.0-20160804073959-5bb0b2dea91e
	github.com/filecoin-project/chain-validation v0.0.3
	github.com/filecoin-project/filecoin-ffi v0.0.0-20191221090835-c7bbef445934 // indirect
	//github.com/filecoin-project/filecoin-ffi v0.0.0-20191204125133-ebb3e13addf1
	github.com/filecoin-project/lotus v0.1.5
	github.com/ipfs/go-cid v0.0.4
	github.com/libp2p/go-libp2p-core v0.3.0
	github.com/stretchr/testify v1.4.0
	gotest.tools v2.2.0+incompatible
	yuncun.com/main/httptest v0.0.0-00010101000000-000000000000
	yuncun.com/main/rpc v0.0.0-00010101000000-000000000000
	yuncun.com/main/server v0.0.0-00010101000000-000000000000
)

replace yuncun.com/main/rpc => ./main/rpc

replace yuncun.com/main/httptest => ./main/httptest

replace yuncun.com/main/server => ./main/server
