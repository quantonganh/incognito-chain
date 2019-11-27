package peerv2

import (
	p2pgrpc "github.com/incognitochain/go-libp2p-grpc"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Connector struct {
	host      host.Host
	prtc      *p2pgrpc.GRPCProtocol
	bootstrap peer.AddrInfo
}

func NewConnector(host host.Host, prtc *p2pgrpc.GRPCProtocol, bootstrap peer.AddrInfo) *Connector {
	return &Connector{
		host:      host,
		prtc:      prtc,
		bootstrap: bootstrap,
	}
}
