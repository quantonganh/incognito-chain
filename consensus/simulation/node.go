package main

import (
	"fmt"
	"github.com/incognitochain/incognito-chain/blockchain"
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/consensus/blsbft"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/wire"
	"os"
)

type Node struct {
	consensusEngine *blsbft.BLSBFT
	chain           *Chain
}

var committeePkStruct int

type logWriter struct {
	NodeID string
	fd     *os.File
}

func (s logWriter) Write(p []byte) (n int, err error) {
	s.fd.Write(p)
	return len(p), nil
}

func NewNode(committeePkStruct []incognitokey.CommitteePublicKey, committee []string, index int) *Node {
	node := Node{}
	node.chain = NewChain(committeePkStruct)

	fd, err := os.OpenFile(fmt.Sprintf("node-%d.log", index), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	fd.Truncate(0)
	backendLog := common.NewBackend(logWriter{
		NodeID: fmt.Sprintf("node-%d", index),
		fd:     fd,
	})
	logger := backendLog.Logger("Consensus", false)

	node.consensusEngine = &blsbft.BLSBFT{
		Chain:    node.chain,
		Node:     node,
		ChainKey: "shard",
		PeerID:   fmt.Sprintf("node-%d", index),
		Logger:   logger,
	}

	prvSeed, err := blsbft.LoadUserKeyFromIncPrivateKey(committee[index])
	failOnError(err)
	failOnError(node.consensusEngine.LoadUserKey(prvSeed))
	return &node
}

func (s *Node) Start() {
	s.consensusEngine.Start()
}
func (Node) PushMessageToChain(msg wire.Message, chain blockchain.ChainInterface) error {
	panic("implement me")
}

func (Node) UpdateConsensusState(role string, userPbk string, currentShard *byte, beaconCommittee []string, shardCommittee map[byte][]string) {
	return
}

func (Node) IsEnableMining() bool {
	return true
}

func (Node) GetMiningKeys() string {
	panic("implement me")
}

func (Node) GetPrivateKey() string {
	panic("implement me")
}

func (Node) DropAllConnections() {
	return
}
