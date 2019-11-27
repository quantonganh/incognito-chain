package peerv2

import (
	"math/rand"
	"time"

	"google.golang.org/grpc"
)

type Connector interface {
	GetAllConnections() []*grpc.ClientConn
	GetConnectionsToShard(int) []*grpc.ClientConn
}

type Indexer struct {
	indexes []HighwayInfo

	connector Connector
}

func NewIndexer() *Indexer {
	return &Indexer{}
}

func (idxer *Indexer) Start() {
	for ; true; <-time.Tick(30 * time.Minute) {
		conns := idxer.GetAllConnections()
		conn := conns[rand.Intn(len(conns))]
	}
}
