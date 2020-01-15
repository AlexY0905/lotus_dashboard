module server

go 1.13

require (
	github.com/filecoin-project/lotus v0.1.5
	github.com/gorilla/websocket v1.4.1
	yuncun.com/main/rpc v0.0.0-00010101000000-000000000000
)

replace yuncun.com/main/rpc => ../rpc
